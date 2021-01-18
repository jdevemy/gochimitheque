package datastores

import (
	"database/sql"

	"github.com/huandu/go-sqlbuilder"
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
		parent StoreLocation
		sb     *sqlbuilder.SelectBuilder
	)

	globals.Log.WithFields(logrus.Fields{"s": s}).Debug("buildFullPath")

	// Recursively getting the parents.
	if s.StoreLocation != nil && s.StoreLocation.StoreLocationID.Valid {

		sb = sqlbuilder.NewSelectBuilder()

		sb.Select("s.storelocation_id",
			"s.storelocation_name",
			sb.As("storelocation.storelocation_id", "storelocation.storelocation_id"),
			sb.As("storelocation.storelocation_name", "storelocation.storelocation_name"),
		)
		sb.From(sb.As("storelocation", "s"))
		sb.JoinWithOption(sqlbuilder.LeftJoin,
			"storelocation",
			sb.Equal("s.storelocation", "storelocation.storelocation_id"),
		)
		sb.Where(sb.Equal("s.storelocation_id", s.StoreLocation.StoreLocationID.Int64))
		sql, args := sb.Build()

		if err = tx.Get(&parent, sql, args...); err != nil {
			globals.Log.Error(err)
			return ""
		}

		return db.buildFullPath(parent, tx) + "/" + s.StoreLocationName.String

	}

	return s.StoreLocationName.String

}

// GetStoreLocations return the store locations matching the p search criteria.
func (db *SQLiteDataStore) GetStoreLocations(p DbselectparamStoreLocation) ([]StoreLocation, int, error) {

	var (
		err                         error
		storelocations              []StoreLocation
		count                       int
		countBuilder, selectBuilder *sqlbuilder.SelectBuilder
	)

	globals.Log.WithFields(logrus.Fields{"p": p}).Debug("GetStoreLocations")

	// Named statements.
	personid := sql.Named("personid", p.GetLoggedPersonID())
	entity := sql.Named("entity", p.GetEntity())
	storelocation_canstore := sql.Named("storelocation_canstore", p.GetStoreLocationCanStore())
	permission := sql.Named("permission", p.GetPermission())
	search := sql.Named("search", p.GetSearch())

	// Select and count.
	countBuilder = sqlbuilder.NewSelectBuilder()
	selectBuilder = sqlbuilder.NewSelectBuilder()

	countBuilder.Select("COUNT(s.storelocation_id)").Distinct()
	selectBuilder.Select(selectBuilder.As("s.storelocation_id", "storelocation_id"),
		selectBuilder.As("s.storelocation_name", "storelocation_name"),
		selectBuilder.As("s.storelocation_fullpath", "storelocation_fullpath"),
		selectBuilder.As("entity.entity_id", "entity.entity_id"),
		selectBuilder.As("entity.entity_name", "entity.entity_name"),
		"s.storelocation_canstore",
		"s.storelocation_color",
	)

	// From.
	fromBuilder := sqlbuilder.NewSelectBuilder()
	fromBuilder.From(fromBuilder.As("storelocation", "s"))
	fromBuilder.Join("entity", fromBuilder.Equal("s.entity", "entity.entity_id"))
	fromBuilder.JoinWithOption(sqlbuilder.LeftJoin, "storelocation", fromBuilder.Equal("s.storelocation", "storelocation.storelocation_id"))
	fromBuilder.Join(fromBuilder.As("permission", "perm"),
		fromBuilder.And(fromBuilder.Equal("perm.person", personid),
			fromBuilder.In("perm.permission_item_name", "all", "storages"),
			fromBuilder.In("perm.permission_perm_name", "all", permission),
			fromBuilder.In("perm.permission_entity_id", -1, "entity.entity_id"),
		))

	// Where.
	whereBuilder := sqlbuilder.NewSelectBuilder()
	whereAndExpression := []string{whereBuilder.Like("s.storelocation_name", search)}
	if p.GetEntity() != -1 {
		whereAndExpression = append(whereAndExpression, whereBuilder.Equal("s.entity", entity))
	}
	if p.GetStoreLocationCanStore() {
		whereAndExpression = append(whereAndExpression, whereBuilder.Equal("s.storelocation_canstore", storelocation_canstore))
	}
	whereBuilder.Where(whereBuilder.And(whereAndExpression...))

	// Order by, group by, limit, offset.
	postBuilder := sqlbuilder.NewSelectBuilder()
	postBuilder.GroupBy("s.storelocation_id")
	postBuilder.OrderBy(p.GetOrderBy(), p.GetOrder())
	postBuilder.Limit(int(p.GetLimit()))
	postBuilder.Offset(int(p.GetOffset()))

	sqlSelect, argsSelect := sqlbuilder.Build("$? $? $? $?", selectBuilder, fromBuilder, whereBuilder, postBuilder).Build()
	sqlCount, argsCount := sqlbuilder.Build("$? $? $?", countBuilder, fromBuilder, whereBuilder).Build()

	globals.Log.Debug(sqlSelect)
	globals.Log.Debug(argsSelect)
	// precreq.WriteString(" SELECT count(DISTINCT s.storelocation_id)")
	// presreq.WriteString(` SELECT s.storelocation_id AS "storelocation_id",
	// s.storelocation_name AS "storelocation_name",
	// s.storelocation_canstore,
	// s.storelocation_color,
	// s.storelocation_fullpath AS "storelocation_fullpath",
	// storelocation.storelocation_id AS "storelocation.storelocation_id",
	// storelocation.storelocation_name AS "storelocation.storelocation_name",
	// entity.entity_id AS "entity.entity_id",
	// entity.entity_name AS "entity.entity_name"`)
	// comreq.WriteString(" FROM storelocation AS s")
	// comreq.WriteString(" JOIN entity ON s.entity = entity.entity_id")
	// comreq.WriteString(" LEFT JOIN storelocation on s.storelocation = storelocation.storelocation_id")

	// // filter by permissions
	// comreq.WriteString(` JOIN permission AS perm ON
	// perm.person = :personid and (perm.permission_item_name in ("all", "storages")) and (perm.permission_perm_name in ("all", :permission)) and (perm.permission_entity_id in (-1, entity.entity_id))
	// `)
	// comreq.WriteString(" WHERE s.storelocation_name LIKE :search")
	// if p.GetEntity() != -1 {
	// 	comreq.WriteString(" AND s.entity = :entity")
	// }
	// if p.GetStoreLocationCanStore() {
	// 	comreq.WriteString(" AND s.storelocation_canstore = :storelocation_canstore")
	// }
	// postsreq.WriteString(" GROUP BY s.storelocation_id")
	// postsreq.WriteString(" ORDER BY " + p.GetOrderBy() + " " + p.GetOrder())

	// // limit
	// if p.GetLimit() != ^uint64(0) {
	// 	postsreq.WriteString(" LIMIT :limit OFFSET :offset")
	// }

	// // building count and select statements
	// if cnstmt, err = db.PrepareNamed(precreq.String() + comreq.String()); err != nil {
	// 	return nil, 0, err
	// }
	// if snstmt, err = db.PrepareNamed(presreq.String() + comreq.String() + postsreq.String()); err != nil {
	// 	return nil, 0, err
	// }

	// // building argument map
	// m := map[string]interface{}{
	// 	"search":                 p.GetSearch(),
	// 	"storelocation_canstore": p.GetStoreLocationCanStore(),
	// 	"personid":               p.GetLoggedPersonID(),
	// 	"order":                  p.GetOrder(),
	// 	"limit":                  p.GetLimit(),
	// 	"offset":                 p.GetOffset(),
	// 	"entity":                 p.GetEntity(),
	// 	"permission":             p.GetPermission(),
	// }
	//globals.Log.Debug(presreq.String() + comreq.String() + postsreq.String())
	//globals.Log.Debug(m)

	// select
	// if err = snstmt.Select(&storelocations, m); err != nil {
	// 	return nil, 0, err
	// }
	// // count
	// if err = cnstmt.Get(&count, m); err != nil {
	// 	return nil, 0, err
	// }

	if err = db.Select(&storelocations, sqlSelect, argsSelect...); err != nil {
		globals.Log.Error(err)
		return nil, 0, err
	}
	if err = db.Get(&count, sqlCount, argsCount...); err != nil {
		globals.Log.Error(err)
		return nil, 0, err
	}

	return storelocations, count, nil

}

// GetStoreLocation returns the store location with id "id"
func (db *SQLiteDataStore) GetStoreLocation(id int) (StoreLocation, error) {
	var (
		storelocation StoreLocation
		sqlr          string
		err           error
	)
	globals.Log.WithFields(logrus.Fields{"id": id}).Debug("GetStoreLocation")

	sqlr = `SELECT s.storelocation_id, s.storelocation_name, s.storelocation_canstore, s.storelocation_color, s.storelocation_fullpath,
	storelocation.storelocation_id AS "storelocation.storelocation_id",
	storelocation.storelocation_name AS "storelocation.storelocation_name",
	entity.entity_id AS "entity.entity_id",
	entity.entity_name AS "entity.entity_name"
	FROM storelocation AS s
	JOIN entity ON s.entity = entity.entity_id
	LEFT JOIN storelocation on s.storelocation = storelocation.storelocation_id
	WHERE s.storelocation_id = ?`
	if err = db.Get(&storelocation, sqlr, id); err != nil {
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

// CreateStoreLocation insert the storelocation s into the DB.
func (db *SQLiteDataStore) CreateStoreLocation(s StoreLocation) (int64, error) {

	var (
		err           error
		tx            *sqlx.Tx
		sqlResult     sql.Result
		insertColumns []string
		insertValues  []interface{}
		insertBuilder *sqlbuilder.InsertBuilder
	)

	if tx, err = db.Beginx(); err != nil {
		return 0, nil
	}

	insertBuilder = sqlbuilder.NewInsertBuilder()

	// Buiding columns and values.
	if s.StoreLocationCanStore.Valid {
		insertColumns = append(insertColumns, "storelocation_canstore")
		insertValues = append(insertValues, s.StoreLocationCanStore.Bool)
	}
	if s.StoreLocationColor.Valid {
		insertColumns = append(insertColumns, "storelocation_color")
		insertValues = append(insertValues, s.StoreLocationColor.String)
	}
	if s.StoreLocation != nil {
		insertColumns = append(insertColumns, "storelocation")
		insertValues = append(insertValues, s.StoreLocation.StoreLocationID.Int64)
	}
	insertColumns = append(insertColumns, "entity")
	insertValues = append(insertValues, s.EntityID)
	insertColumns = append(insertColumns, "storelocation_fullpath")
	insertValues = append(insertValues, db.buildFullPath(s, tx))

	// Buiding the query.
	insertBuilder.InsertInto("storelocation")
	insertBuilder.Cols(insertColumns...)
	insertBuilder.Values(insertValues...)
	sql, args := insertBuilder.Build()

	if sqlResult, err = tx.Exec(sql, args...); err != nil {
		_ = tx.Rollback()
		return 0, nil
	}

	if err = tx.Commit(); err != nil {
		return 0, nil
	}

	return sqlResult.LastInsertId()

}

// UpdateStoreLocation update the storelocation s into the DB.
func (db *SQLiteDataStore) UpdateStoreLocation(s StoreLocation) error {

	var (
		err           error
		tx            *sqlx.Tx
		updateBuilder *sqlbuilder.UpdateBuilder
	)

	if tx, err = db.Beginx(); err != nil {
		return err
	}

	updateBuilder = sqlbuilder.NewUpdateBuilder()

	// Buiding columns and values.
	assignments := []string{}
	if s.StoreLocationCanStore.Valid {
		assignments = append(assignments, updateBuilder.Assign("storelocation_canstore", s.StoreLocationCanStore.Bool))
	}
	if s.StoreLocationColor.Valid {
		assignments = append(assignments, updateBuilder.Assign("storelocation_color", s.StoreLocationColor.String))
	}
	if s.StoreLocation != nil {
		assignments = append(assignments, updateBuilder.Assign("storelocation", s.StoreLocation.StoreLocationID.Int64))
	}
	assignments = append(assignments, updateBuilder.Assign("storelocation_name", s.StoreLocationName.String))
	assignments = append(assignments, updateBuilder.Assign("entity", s.EntityID))
	assignments = append(assignments, updateBuilder.Assign("storelocation_fullpath", db.buildFullPath(s, tx)))

	// Buiding the query.
	updateBuilder.Update("storelocation")
	updateBuilder.Set(assignments...)
	updateBuilder.Where(updateBuilder.Equal("storelocation_id", s.StoreLocationID))
	sql, args := updateBuilder.Build()

	if _, err = tx.Exec(sql, args...); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()

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
