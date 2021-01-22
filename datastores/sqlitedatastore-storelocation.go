package datastores

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/tbellembois/gochimitheque/globals"
	. "github.com/tbellembois/gochimitheque/models"
)

// Return the store location full path.
// The caller is responsible of opening and commiting the tx transaction.
func (db *SQLiteDataStore) buildFullPath(s StoreLocation, tx *sqlx.Tx) string {

	var (
		err    error
		sqlr   string
		args   []interface{}
		parent StoreLocation
	)

	globals.Log.WithFields(logrus.Fields{"s": s}).Debug("buildFullPath")

	// Recursively getting the parents.
	if s.StoreLocation != nil && s.StoreLocation.StoreLocationID.Valid {

		dialect := goqu.Dialect("sqlite3")
		t := goqu.T("storelocation")

		sQuery := dialect.From(t.As("s")).Select(
			goqu.I("s.storelocation_id"),
			goqu.I("s.storelocation_name"),
			goqu.I("storelocation.storelocation_id").As(goqu.C("storelocation.storelocation_id")),
			goqu.I("storelocation.storelocation_name").As(goqu.C("storelocation.storelocation_name")),
		).LeftJoin(
			goqu.T("storelocation"),
			goqu.On(goqu.Ex{
				"s.storelocation": goqu.I("storelocation.storelocation_id"),
			}),
		).Where(
			goqu.I("s.storelocation_id").Eq(s.StoreLocation.StoreLocationID.Int64),
		)

		if sqlr, args, err = sQuery.ToSQL(); err != nil {
			globals.Log.Error(err)
			return ""
		}

		if err = tx.Get(&parent, sqlr, args...); err != nil {
			globals.Log.Error(err)
			return ""
		}

		return db.buildFullPath(parent, tx) + "/" + s.StoreLocationName.String

	}

	return s.StoreLocationName.String

}

// GetStoreLocations select the store locations matching p.
func (db *SQLiteDataStore) GetStoreLocations(p DbselectparamStoreLocation) ([]StoreLocation, int, error) {

	var (
		err                   error
		storelocations        []StoreLocation
		count                 int
		countSql, selectSql   string
		countArgs, selectArgs []interface{}
	)

	globals.Log.WithFields(logrus.Fields{"p": p}).Debug("GetStoreLocations")

	dialect := goqu.Dialect("sqlite3")
	t := goqu.T("storelocation")

	// Map orderby clause.
	orderByClause := p.GetOrderBy()
	if orderByClause == "storelocation" {
		orderByClause = "storelocation.storelocation_id"
	}

	// Prepare orderby/order clause.
	orderClause := goqu.I(orderByClause).Asc()
	if strings.ToLower(p.GetOrder()) == "desc" {
		orderClause = goqu.I(orderByClause).Desc()
	}

	// Select and join.
	selectClause := dialect.From(t.As("s")).Join(
		goqu.T("entity"),
		goqu.On(goqu.Ex{"s.entity": goqu.I("entity.entity_id")}),
	).LeftJoin(
		goqu.T("storelocation"),
		goqu.On(goqu.Ex{"s.storelocation": goqu.I("storelocation.storelocation_id")}),
	).Join(
		goqu.T("permission").As("perm"),
		goqu.On(
			goqu.Ex{
				"perm.person":               p.GetLoggedPersonID(),
				"perm.permission_item_name": []string{"all", "storages"},
				"perm.permission_perm_name": []string{"all", p.GetPermission()},
				"perm.permission_entity_id": []interface{}{-1, goqu.I("entity.entity_id")},
			},
		),
	)

	// Where.
	whereAnd := []goqu.Expression{
		goqu.I("s.storelocation_name").Like(p.GetSearch()),
	}
	if p.GetEntity() != -1 {
		whereAnd = append(whereAnd, goqu.I("s.entity").Eq(p.GetEntity()))
	}
	if p.GetStoreLocationCanStore() {
		whereAnd = append(whereAnd, goqu.I("s.storelocation_canstore").Eq(p.GetStoreLocationCanStore()))
	}
	selectClause = selectClause.Where(goqu.And(whereAnd...))

	if countSql, countArgs, err = selectClause.Select(
		goqu.COUNT(goqu.I("s.storelocation_id").Distinct()),
	).ToSQL(); err != nil {
		return nil, 0, err
	}
	if selectSql, selectArgs, err = selectClause.Select(
		goqu.I("s.storelocation_id").As("storelocation_id"),
		goqu.I("s.storelocation_canstore").As("storelocation_canstore"),
		goqu.I("s.storelocation_color").As("storelocation_color"),
		goqu.I("s.storelocation_id").As("storelocation_id"),
		goqu.I("s.storelocation_name").As("storelocation_name"),
		goqu.I("s.storelocation_fullpath").As("storelocation_fullpath"),
		goqu.I("storelocation.storelocation_id").As(goqu.C("storelocation.storelocation_id")),
		goqu.I("storelocation.storelocation_name").As(goqu.C("storelocation.storelocation_name")),
		goqu.I("entity.entity_id").As(goqu.C("entity.entity_id")),
		goqu.I("entity.entity_name").As(goqu.C("entity.entity_name")),
	).GroupBy(goqu.I("s.storelocation_id")).Order(orderClause).Limit(uint(p.GetLimit())).Offset(uint(p.GetOffset())).ToSQL(); err != nil {
		return nil, 0, err
	}

	// globals.Log.Debug(selectSql)
	// globals.Log.Debug(selectArgs)
	// globals.Log.Debug(countSql)
	// globals.Log.Debug(countArgs)

	// select
	if err = db.Select(&storelocations, selectSql, selectArgs...); err != nil {
		return nil, 0, err
	}
	// count
	if err = db.Get(&count, countSql, countArgs...); err != nil {
		return nil, 0, err
	}

	return storelocations, count, nil

}

// GetStoreLocation select the store location by id.
func (db *SQLiteDataStore) GetStoreLocation(id int) (StoreLocation, error) {

	var (
		err           error
		sqlr          string
		args          []interface{}
		storelocation StoreLocation
	)

	globals.Log.WithFields(logrus.Fields{"id": id}).Debug("GetStoreLocation")

	dialect := goqu.Dialect("sqlite3")
	t := goqu.T("storelocation")

	sQuery := dialect.From(t.As("s")).Join(
		goqu.T("entity"),
		goqu.On(goqu.Ex{"s.entity": goqu.I("entity.entity_id")}),
	).LeftJoin(
		goqu.T("storelocation"),
		goqu.On(goqu.Ex{"s.storelocation": goqu.I("storelocation.storelocation_id")}),
	).Where(
		goqu.I("s.storelocation_id").Eq(id),
	).Select(
		goqu.I("s.storelocation_id"),
		goqu.I("s.storelocation_name"),
		goqu.I("s.storelocation_canstore"),
		goqu.I("s.storelocation_color"),
		goqu.I("s.storelocation_fullpath"),
		goqu.I("storelocation.storelocation_id").As(goqu.C("storelocation.storelocation_id")),
		goqu.I("storelocation.storelocation_name").As(goqu.C("storelocation.storelocation_name")),
		goqu.I("entity.entity_id").As(goqu.C("entity.entity_id")),
		goqu.I("entity.entity_name").As(goqu.C("entity.entity_name")),
	)

	if sqlr, args, err = sQuery.ToSQL(); err != nil {
		globals.Log.Error(err)
		return StoreLocation{}, err
	}

	// globals.Log.Debug(sql)
	// globals.Log.Debug(args)

	if err = db.Get(&storelocation, sqlr, args...); err != nil {
		return StoreLocation{}, err
	}

	globals.Log.WithFields(logrus.Fields{"ID": id, "storelocation": storelocation}).Debug("GetStoreLocation")
	return storelocation, nil

}

// GetStoreLocationChildren select the children store locations of parent id.
func (db *SQLiteDataStore) GetStoreLocationChildren(id int) ([]StoreLocation, error) {

	var (
		err            error
		sqlr           string
		args           []interface{}
		storelocations []StoreLocation
	)

	dialect := goqu.Dialect("sqlite3")
	t := goqu.T("storelocation")

	// Select
	sQuery := dialect.From(t.As("s")).Select(
		goqu.I("s.storelocation_id"),
		goqu.I("s.storelocation_name"),
		goqu.I("s.storelocation_canstore"),
		goqu.I("s.storelocation_color"),
		goqu.I("s.storelocation_fullpath"),
		goqu.I("storelocation.storelocation_id").As(goqu.C("storelocation.storelocation_id")),
		goqu.I("storelocation.storelocation_name").As(goqu.C("storelocation.storelocation_name")),
		goqu.I("entity.entity_id").As(goqu.C("entity.entity_id")),
		goqu.I("entity.entity_name").As(goqu.C("entity.entity_name")),
	).Join(
		goqu.T("entity"),
		goqu.On(goqu.Ex{"s.entity": goqu.I("entity.entity_id")}),
	).LeftJoin(
		goqu.T("storelocation"),
		goqu.On(goqu.Ex{"s.storelocation": goqu.I("storelocation.storelocation_id")}),
	).Where(
		goqu.I("s.storelocation").Eq(id),
	)

	if sqlr, args, err = sQuery.ToSQL(); err != nil {
		globals.Log.Error(err)
		return nil, err
	}

	if err = db.Select(&storelocations, sqlr, args...); err != nil {
		return nil, err
	}

	return storelocations, nil

}

// GetStoreLocationEntity select the store location entity.
func (db *SQLiteDataStore) GetStoreLocationEntity(id int) (Entity, error) {

	var (
		err    error
		sqlr   string
		args   []interface{}
		entity Entity
	)

	globals.Log.WithFields(logrus.Fields{"id": id}).Debug("GetStoreLocationEntity")

	dialect := goqu.Dialect("sqlite3")
	t := goqu.T("storelocation")

	sQuery := dialect.From(t.As("s")).Join(
		goqu.T("entity"),
		goqu.On(goqu.Ex{"s.entity": goqu.I("entity.entity_id")}),
	).Where(
		goqu.I("s.storelocation_id").Eq(id),
	).Select(
		goqu.I("entity.entity_id").As("entity_id"),
		goqu.I("entity.entity_name").As("entity_name"),
	)

	if sqlr, args, err = sQuery.ToSQL(); err != nil {
		globals.Log.Error(err)
		return Entity{}, err
	}

	if err = db.Get(&entity, sqlr, args...); err != nil {
		return Entity{}, err
	}

	return entity, nil

}

// DeleteStoreLocation delete the store location by id.
func (db *SQLiteDataStore) DeleteStoreLocation(id int) error {

	var (
		err  error
		sqlr string
		args []interface{}
	)

	dialect := goqu.Dialect("sqlite3")
	t := goqu.T("storelocation")

	sQuery := dialect.From(t).Where(
		goqu.I("storelocation_id").Eq(id),
	).Delete()

	if sqlr, args, err = sQuery.ToSQL(); err != nil {
		globals.Log.Error(err)
		return err
	}

	if _, err = db.Exec(sqlr, args...); err != nil {
		return err
	}

	return nil

}

// CreateStoreLocation insert s.
func (db *SQLiteDataStore) CreateStoreLocation(s StoreLocation) (int64, error) {

	var (
		err  error
		sqlr string
		args []interface{}
		res  sql.Result
		tx   *sqlx.Tx
	)

	globals.Log.WithFields(logrus.Fields{"s": fmt.Sprintf("%+v", s)}).Debug("CreateStoreLocation")

	dialect := goqu.Dialect("sqlite3")
	t := goqu.T("storelocation")

	if tx, err = db.Beginx(); err != nil {
		return 0, nil
	}

	s.StoreLocationFullPath = db.buildFullPath(s, tx)

	iQuery := dialect.Insert(t)

	setClause := goqu.Record{
		"storelocation_name":     s.StoreLocationName.String,
		"entity":                 s.EntityID,
		"storelocation_fullpath": s.StoreLocationFullPath,
	}

	if s.StoreLocationCanStore.Valid {
		setClause["storelocation_canstore"] = s.StoreLocationCanStore.Bool
	}
	if s.StoreLocationColor.Valid {
		setClause["storelocation_color"] = s.StoreLocationColor.String
	}
	if s.StoreLocation != nil {
		setClause["storelocation"] = s.StoreLocation.StoreLocationID.Int64
	}

	if sqlr, args, err = iQuery.Rows(setClause).ToSQL(); err != nil {
		globals.Log.Error(err)
		return 0, err
	}

	if res, err = tx.Exec(sqlr, args...); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, nil
	}

	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, nil
	}

	return res.LastInsertId()

}

// UpdateStoreLocation updates s.
func (db *SQLiteDataStore) UpdateStoreLocation(s StoreLocation) error {

	var (
		err  error
		sqlr string
		args []interface{}
		tx   *sqlx.Tx
	)

	dialect := goqu.Dialect("sqlite3")
	t := goqu.T("storelocation")

	if tx, err = db.Beginx(); err != nil {
		return err
	}

	s.StoreLocationFullPath = db.buildFullPath(s, tx)

	uQuery := dialect.Update(t)

	setClause := goqu.Record{
		"storelocation_name":     s.StoreLocationName.String,
		"entity":                 s.EntityID,
		"storelocation_fullpath": s.StoreLocationFullPath,
	}

	if s.StoreLocationCanStore.Valid {
		setClause["storelocation_canstore"] = s.StoreLocationCanStore.Bool
	}
	if s.StoreLocationColor.Valid {
		setClause["storelocation_color"] = s.StoreLocationColor.String
	}
	if s.StoreLocation != nil {
		setClause["storelocation"] = s.StoreLocation.StoreLocationID.Int64
	}

	if sqlr, args, err = uQuery.Set(
		setClause,
	).Where(
		goqu.I("storelocation_id").Eq(s.StoreLocationID),
	).ToSQL(); err != nil {
		globals.Log.Error(err)
		return err
	}

	if _, err = tx.Exec(sqlr, args...); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
		return err
	}

	return nil

}

// IsStoreLocationEmpty returns true is the store location is empty.
func (db *SQLiteDataStore) IsStoreLocationEmpty(id int) (bool, error) {

	var (
		err   error
		sqlr  string
		args  []interface{}
		count int
	)

	dialect := goqu.Dialect("sqlite3")
	t := goqu.T("storage")

	sQuery := dialect.From(t).Select(
		goqu.COUNT("*"),
	).Where(
		goqu.I("storelocation").Eq(id),
	)

	if sqlr, args, err = sQuery.ToSQL(); err != nil {
		globals.Log.Error(err)
		return false, err
	}

	if err = db.Get(&count, sqlr, args...); err != nil {
		return false, err
	}

	return count == 0, nil

}
