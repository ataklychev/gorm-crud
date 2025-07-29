package gorm_crud

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type WithTableName interface {
	TableName() string
}

type ListParametersInterface interface{}

type PaginationParameters struct {
	Page      int    `form:"page,default=0" json:"page,default=0"`
	PageSize  int    `form:"page_size,default=30" json:"page_size,default=30"`
	OrderBy   string `form:"order_by,default=id" json:"order_by,default=id"`
	OrderDesc bool   `form:"order_desc,default=false" json:"order_desc,default=false"`
}

type CrudListParameters struct {
	*PaginationParameters
}

const DefaultPageSize = 30

type ListQueryBuilderInterface interface {
	ListQuery(parameters ListParametersInterface) (*gorm.DB, error)
}

type BaseListQueryBuilder struct {
	Db     *gorm.DB
	Logger LoggerInterface
	ListQueryBuilderInterface
}

func NewBaseListQueryBuilder(db *gorm.DB, logger LoggerInterface) *BaseListQueryBuilder {
	return &BaseListQueryBuilder{Db: db, Logger: logger}
}

func (c BaseListQueryBuilder) paginationQuery(parameters ListParametersInterface) *gorm.DB {
	query := c.Db

	val := reflect.ValueOf(parameters).Elem()
	if val.Kind() != reflect.Struct {
		c.Logger.Error("gorm-crud: Unexpected type of parameters for paginationQuery")
		return query
	}

	paginationParameters := val.FieldByName("PaginationParameters")
	hasPaginationParams := paginationParameters.IsValid() && !paginationParameters.IsNil()

	var page int64
	page = 0
	if hasPaginationParams {
		pageValue := val.FieldByName("Page")
		if !pageValue.IsValid() || pageValue.Kind() != reflect.Int {
			c.Logger.Error("gorm-crud: Page is not specified correctly in listQuery")
		} else {
			page = pageValue.Int()
		}
	}

	var pageSize int64
	pageSize = DefaultPageSize
	if hasPaginationParams {
		pageSizeValue := val.FieldByName("PageSize")
		if !pageSizeValue.IsValid() || pageSizeValue.Kind() != reflect.Int {
			c.Logger.Error("gorm-crud: PageSize is not specified in listQuery")
		} else {
			pageSize = pageSizeValue.Int()
		}
	}

	limit := pageSize
	offset := page * pageSize
	query = query.Offset(int(offset)).Limit(int(limit))

	var orderBy string
	if hasPaginationParams {
		orderByValue := val.FieldByName("OrderBy")
		if orderByValue.IsValid() && orderByValue.Kind() == reflect.String {
			orderBy = orderByValue.String()
		}
	}

	var orderDesc = false
	if hasPaginationParams {
		orderDescValue := val.FieldByName("OrderDesc")
		if orderDescValue.IsValid() && orderDescValue.Kind() == reflect.Bool {
			orderDesc = orderDescValue.Bool()
		}
	}

	if len(orderBy) > 0 {
		if orderDesc {
			query = query.Order(fmt.Sprintf("%s DESC", orderBy))
		} else {
			query = query.Order(fmt.Sprintf("%s ASC", orderBy))
		}
	}

	return query
}

func (c BaseListQueryBuilder) ListQuery(parameters ListParametersInterface) (*gorm.DB, error) {
	return c.paginationQuery(parameters), nil
}

type CrudRepositoryInterface interface {
	BaseRepositoryInterface
	GetModel() InterfaceEntity
	Find(id uint) (InterfaceEntity, error)
	PluckBy(fieldNames []string) (map[string]int64, error)
	ListAll() ([]InterfaceEntity, error)
	List(parameters ListParametersInterface) ([]InterfaceEntity, error)
	ListCount(parameters ListParametersInterface) (int64, error)
	Create(item InterfaceEntity) InterfaceEntity
	CreateOrUpdateMany(item InterfaceEntity, columns []string, values []map[string]interface{}, onConflict string) error
	Update(item InterfaceEntity) InterfaceEntity
	Delete(id uint) error
}

type CrudRepository struct {
	CrudRepositoryInterface
	*BaseRepository
	Model            InterfaceEntity // Dynamic typing
	ListQueryBuilder ListQueryBuilderInterface
}

func NewCrudRepository(db *gorm.DB, model InterfaceEntity, listQueryBuilder ListQueryBuilderInterface, logger LoggerInterface) *CrudRepository {
	repo := NewBaseRepository(db, logger)
	return &CrudRepository{
		BaseRepository:   repo,
		Model:            model,
		ListQueryBuilder: listQueryBuilder,
	}
}

func (c CrudRepository) GetModel() InterfaceEntity {
	return c.Model
}

func (c CrudRepository) Find(id uint) (InterfaceEntity, error) {
	item := reflect.New(reflect.TypeOf(c.GetModel()).Elem()).Interface()
	err := c.Db.First(item, id).Error
	return item, NormalizeErr(err)
}

func (c CrudRepository) PluckBy(fieldNames []string) (map[string]int64, error) {

	res := map[string]int64{}

	items, err := c.ListAll()
	if nil != err {
		return res, err
	}

	for _, item := range items {

		// build key
		values := make([]string, 0)
		val := reflect.ValueOf(item)
		for _, fieldName := range fieldNames {
			if val.FieldByName(fieldName).IsValid() {
				values = append(values, val.FieldByName(fieldName).String())
			} else {
				return res, fmt.Errorf("field with name (%s) does not exists on entity (%s)", fieldName, reflect.TypeOf(item))
			}
		}

		pluckKey := strings.Join(values, "_")

		res[pluckKey] = Num64(val.FieldByName("ID").Interface())
	}

	return res, err
}

func (c CrudRepository) ListAll() ([]InterfaceEntity, error) {
	entities := make([]InterfaceEntity, 0)

	page := 0
	pageSize := 10000

	for {
		parameters := new(CrudListParameters)
		parameters.PaginationParameters = new(PaginationParameters)
		parameters.OrderBy = "id"
		parameters.OrderDesc = false
		parameters.PageSize = pageSize
		parameters.Page = page

		items, err := c.List(parameters)
		if nil != err {
			return entities, err
		}

		for _, item := range items {
			entities = append(entities, item)
		}

		if len(items) < pageSize {
			break
		}

		page += 1
	}

	return entities, nil
}

func (c CrudRepository) List(parameters ListParametersInterface) ([]InterfaceEntity, error) {

	items := reflect.New(reflect.SliceOf(reflect.TypeOf(c.GetModel()).Elem())).Interface()
	query, err := c.ListQueryBuilder.ListQuery(parameters)
	if err != nil {
		return []InterfaceEntity{}, err
	}

	err = query.Find(items).Error

	entities := reflect.ValueOf(items).Elem().Interface()

	// Convert entities to slice
	var data []InterfaceEntity
	sliceValue := reflect.ValueOf(entities)
	for i := 0; i < sliceValue.Len(); i++ {
		data = append(data, sliceValue.Index(i).Interface())
	}

	return data, NormalizeErr(err)
}

func (c CrudRepository) ListCount(parameters ListParametersInterface) (int64, error) {
	query, err := c.ListQueryBuilder.ListQuery(parameters)
	if err != nil {
		return 0, err
	}
	var count int64
	item := reflect.New(reflect.TypeOf(c.GetModel()).Elem()).Interface()
	err = query.Model(item).Count(&count).Error
	return count, err
}

func (c CrudRepository) Create(item InterfaceEntity) InterfaceEntity {
	c.Db.Create(item)
	return item
}

func (c *CrudRepository) quote(str string) string {
	// postgres style escape
	str = strings.ReplaceAll(str, "'", "''")
	return fmt.Sprintf("'%s'", str)
}

func (c CrudRepository) prepareTime(val time.Time) string {
	return fmt.Sprintf("'%s'", val.Format("2006-01-02T15:04:05-0700"))
}

func (c CrudRepository) prepareSliceOfNumbers(values interface{}) string {
	result := "{}"
	switch reflect.TypeOf(values).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(values)
		valuesText := []string{}
		for i := 0; i < s.Len(); i++ {
			text := fmt.Sprint(s.Index(i))
			valuesText = append(valuesText, text)
		}
		result = fmt.Sprintf("{%s}", strings.Join(valuesText, ","))
	}

	return c.quote(result)
}

// CreateOrUpdateMany create or update if exists
func (c CrudRepository) CreateOrUpdateMany(
	item InterfaceEntity,
	columns []string,
	values []map[string]interface{},
	onConflict string,
) error {

	if len(values) == 0 {
		return nil
	}

	var valueStrings []string
	for _, valueMap := range values {
		var valueRowString []string
		for _, column := range columns {

			colVal, ok := valueMap[column]
			if !ok {
				return errors.New(fmt.Sprintf("CreateOrUpdateMany: value for column %s found", column))
			}

			// stringify column value
			val := fmt.Sprintf("%v", colVal)

			// filter column value
			switch v := colVal.(type) {
			case sql.NullInt32:
				if !v.Valid {
					val = "NULL"
				} else {
					val = strconv.FormatInt(int64(v.Int32), 10)
				}
			case sql.NullInt64:
				if !v.Valid {
					val = "NULL"
				} else {
					val = strconv.FormatInt(v.Int64, 10)
				}
			case sql.NullFloat64:
				if !v.Valid {
					val = "NULL"
				} else {
					val = fmt.Sprintf("%g", v.Float64)
				}
			case sql.NullBool:
				if !v.Valid {
					val = "NULL"
				} else if v.Bool {
					val = "TRUE"
				} else {
					val = "FALSE"
				}
			case sql.NullTime:
				if !v.Valid {
					val = "NULL"
				} else {
					val = c.prepareTime(v.Time)
				}
			case sql.NullString:
				if !v.Valid {
					val = "NULL"
				} else {
					val = c.quote(v.String)
				}
			case time.Time:
				val = c.prepareTime(colVal.(time.Time))
			case *time.Time:
				if !reflect.ValueOf(colVal).IsNil() {
					t := reflect.ValueOf(colVal).Elem().Interface().(time.Time)
					val = c.prepareTime(t)
				} else {
					val = "NULL"
				}
			case []int64, []int32, []uint8, []float64, []float32, pq.Int64Array, pq.Float64Array:
				val = c.prepareSliceOfNumbers(v)
			default:
				if reflect.TypeOf(colVal).Kind() == reflect.String {
					val = c.quote(val)
				}
			}

			valueRowString = append(valueRowString, val)
		}
		valueString := fmt.Sprintf("(%s)", strings.Join(valueRowString, ","))
		valueStrings = append(valueStrings, valueString)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s %s",
		c.GetTableName(item),
		strings.Join(columns, ","),
		strings.Join(valueStrings, ","),
		onConflict)

	err := c.Db.Exec(query).Error
	err = NormalizeErr(err)
	if nil != err {
		c.Logger.Errorf("gorm-crud: Error in the CreateOrUpdateMany(): %v", err)
	}

	return err
}

func (c CrudRepository) GetTableName(item InterfaceEntity) string {
	if val, ok := item.(WithTableName); ok {
		return val.TableName()
	}
	return c.Db.Model(&item).Statement.Table
}

func (c CrudRepository) Update(item InterfaceEntity) InterfaceEntity {
	c.Db.Save(item)
	return item
}

func (c CrudRepository) Delete(id uint) error {
	item, err := c.Find(id)
	err = NormalizeErr(err)
	if err != nil {
		return err
	}
	c.Db.Delete(item)
	return nil
}
