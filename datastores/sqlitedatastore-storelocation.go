package datastores

import (
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
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
		sql    string
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

		if sql, args, err = sQuery.ToSQL(); err != nil {
			globals.Log.Error(err)
			return ""
		}

		if err = tx.Get(&parent, sql, args...); err != nil {
			globals.Log.Error(err)
			return ""
		}

		return db.buildFullPath(parent, tx) + "/" + s.StoreLocationName.String

	}

	return s.StoreLocationName.String

}

// GetStoreLocations returns the store locations matching the search criteria
// order, offset and limit are passed to the sql request
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

	if countSql, countArgs, err = selectClause.Select(goqu.COUNT(goqu.I("s.storelocation_id").Distinct())).ToSQL(); err != nil {
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

// GetStoreLocation returns the store location with id "id"
func (db *SQLiteDataStore) GetStoreLocation(id int) (StoreLocation, error) {

	var (
		err           error
		sql           string
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

	if sql, args, err = sQuery.ToSQL(); err != nil {
		globals.Log.Error(err)
		return StoreLocation{}, err
	}

	// globals.Log.Debug(sql)
	// globals.Log.Debug(args)

	if err = db.Get(&storelocation, sql, args...); err != nil {
		return StoreLocation{}, err
	}

	globals.Log.WithFields(logrus.Fields{"ID": id, "storelocation": storelocation}).Debug("GetStoreLocation")
	return storelocation, nil

}

// GetStoreLocationChildren returns the children of the store location with id "id"
func (db *SQLiteDataStore) GetStoreLocationChildren(id int) ([]StoreLocation, error) {
	var (
		storelocations []StoreLocation
		sqlr           string
		err            error
	)

	sqlr = `SELECT s.storelocation_id, s.storelocation_name, s.storelocation_canstore, s.storelocation_color, s.storelocation_fullpath,
	storelocation.storelocation_id AS "storelocation.storelocation_id",
	storelocation.storelocation_name AS "storelocation.storelocation_name",
	entity.entity_id AS "entity.entity_id",
	entity.entity_name AS "entity.entity_name"
	FROM storelocation AS s
	JOIN entity ON s.entity = entity.entity_id
	LEFT JOIN storelocation on s.storelocation = storelocation.storelocation_id
	WHERE s.storelocation = ?`
	if err = db.Select(&storelocations, sqlr, id); err != nil {
		return []StoreLocation{}, err
	}

	globals.Log.WithFields(logrus.Fields{"id": id, "storelocations": storelocations}).Debug("GetStoreLocationChildren")
	return storelocations, nil
}

// GetStoreLocationEntity returns the entity of the store location with id "id"
func (db *SQLiteDataStore) GetStoreLocationEntity(id int) (Entity, error) {
	var (
		entity Entity
		sqlr   string
		err    error
	)
	globals.Log.WithFields(logrus.Fields{"id": id}).Debug("GetStoreLocationEntity")

	sqlr = `SELECT 
	entity.entity_id AS "entity_id",
	entity.entity_name AS "entity_name"
	FROM storelocation AS s
	JOIN entity ON s.entity = entity.entity_id
	WHERE s.storelocation_id = ?`
	if err = db.Get(&entity, sqlr, id); err != nil {
		return Entity{}, err
	}
	globals.Log.WithFields(logrus.Fields{"id": id, "entity": entity}).Debug("GetStoreLocationEntity")
	return entity, nil
}

// DeleteStoreLocation deletes the store location with id "id"
func (db *SQLiteDataStore) DeleteStoreLocation(id int) error {
	var (
		sqlr string
		err  error
	)
	sqlr = `DELETE FROM storelocation 
	WHERE storelocation_id = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}
	return nil
}

// CreateStoreLocation creates the given store location
func (db *SQLiteDataStore) CreateStoreLocation(s StoreLocation) (int64, error) {
	var (
		sqlr     string
		res      sql.Result
		lastid   int64
		err      error
		sqla     []interface{}
		tx       *sqlx.Tx
		ibuilder sq.InsertBuilder
	)

	// beginning transaction
	if tx, err = db.Beginx(); err != nil {
		return 0, nil
	}

	// building full path
	s.StoreLocationFullPath = db.buildFullPath(s, tx)

	m := make(map[string]interface{})
	if s.StoreLocationCanStore.Valid {
		m["storelocation_canstore"] = s.StoreLocationCanStore.Bool
	}
	if s.StoreLocationColor.Valid {
		m["storelocation_color"] = s.StoreLocationColor.String
	}
	m["storelocation_name"] = s.StoreLocationName.String
	if s.StoreLocation != nil {
		m["storelocation"] = s.StoreLocation.StoreLocationID.Int64
	}
	m["entity"] = s.EntityID
	m["storelocation_fullpath"] = s.StoreLocationFullPath

	// building column names/values
	col := make([]string, 0, len(m))
	val := make([]interface{}, 0, len(m))
	for k, v := range m {
		col = append(col, k)

		switch t := v.(type) {
		case int:
			val = append(val, v.(int))
		case int64:
			val = append(val, v.(int64))
		case string:
			val = append(val, v.(string))
		case bool:
			val = append(val, v.(bool))
		default:
			panic(fmt.Sprintf("unknown type: %T", t))
		}
	}

	ibuilder = sq.Insert("storelocation").Columns(col...).Values(val...)
	if sqlr, sqla, err = ibuilder.ToSql(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, nil
	}

	if res, err = tx.Exec(sqlr, sqla...); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, nil
	}

	// committing changes
	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, nil
	}

	// getting the last inserted id
	if lastid, err = res.LastInsertId(); err != nil {
		return 0, nil
	}

	return lastid, nil
}

// UpdateStoreLocation updates the given store location
func (db *SQLiteDataStore) UpdateStoreLocation(s StoreLocation) error {
	var (
		sqlr     string
		sqla     []interface{}
		tx       *sqlx.Tx
		err      error
		ubuilder sq.UpdateBuilder
	)

	// beginning new transaction
	if tx, err = db.Beginx(); err != nil {
		return err
	}

	// building full path
	s.StoreLocationFullPath = db.buildFullPath(s, tx)

	m := make(map[string]interface{})
	if s.StoreLocationCanStore.Valid {
		m["storelocation_canstore"] = s.StoreLocationCanStore.Bool
	}
	if s.StoreLocationColor.Valid {
		m["storelocation_color"] = s.StoreLocationColor.String
	}
	m["storelocation_name"] = s.StoreLocationName.String
	if s.StoreLocation != nil {
		m["storelocation"] = s.StoreLocation.StoreLocationID.Int64
	}
	m["entity"] = s.EntityID
	m["storelocation_fullpath"] = s.StoreLocationFullPath

	ubuilder = sq.Update("storelocation").
		SetMap(m).
		Where(sq.Eq{"storelocation_id": s.StoreLocationID})
	if sqlr, sqla, err = ubuilder.ToSql(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
		return err
	}
	if _, err = tx.Exec(sqlr, sqla...); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
		return err
	}

	// committing changes
	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
		return err
	}

	return nil
}

// IsStoreLocationEmpty returns true is the store location is empty
func (db *SQLiteDataStore) IsStoreLocationEmpty(id int) (bool, error) {
	var (
		res   bool
		count int
		sqlr  string
		err   error
	)

	sqlr = "SELECT count(*) from storage WHERE  storelocation = ?"
	if err = db.Get(&count, sqlr, id); err != nil {
		return false, err
	}
	globals.Log.WithFields(logrus.Fields{"id": id, "count": count}).Debug("IsStoreLocationEmpty")
	if count == 0 {
		res = true
	} else {
		res = false
	}
	return res, nil
}
