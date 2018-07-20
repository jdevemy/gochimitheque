package models

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
	log "github.com/sirupsen/logrus"
	"github.com/tbellembois/gochimitheque/constants"
)

// GetPeople returns the people matching the search criteria
// order, offset and limit are passed to the sql request
func (db *SQLiteDataStore) GetPeople(personID int, search string, order string, offset uint64, limit uint64) ([]Person, int, error) {
	var (
		people []Person
		count  int
		sqlr   string
		sqla   []interface{}
	)
	log.WithFields(log.Fields{"search": search, "order": order, "offset": offset, "limit": limit}).Debug("GetPeople")

	// count query
	cbuilder := sq.Select("count(DISTINCT p.person_id)").
		From("person AS p, entity AS e").
		Where("p.person_email LIKE ?", fmt.Sprint("%", search, "%")).
		// join to get person entities
		Join(`personentities ON
			personentities.personentities_person_id = p.person_id`).
		Join(`entity ON
			personentities.personentities_entity_id = e.entity_id`).
		// join to filter people personID can access to
		Join(`permission AS perm on
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "all" and perm.permission_entity_id = e.entity_id) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "all" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "r" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "all" and perm.permission_entity_id = e.entity_id) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "all" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "r" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "r" and perm.permission_entity_id = e.entity_id)
			`, personID, personID, personID, personID, personID, personID, personID)
	// select query
	sbuilder := sq.Select(`p.person_id, 
		p.person_email`).
		From("person AS p, entity AS e").
		Where("p.person_email LIKE ?", fmt.Sprint("%", search, "%")).
		// join to get person entities
		Join(`personentities ON
		personentities.personentities_person_id = p.person_id
		`).
		Join(`entity ON
		personentities.personentities_entity_id = e.entity_id
		`).
		// join to filter people personID can access to
		Join(`permission AS perm on
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "all" and perm.permission_entity_id = e.entity_id) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "all" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "r" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "all" and perm.permission_entity_id = e.entity_id) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "all" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "r" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "r" and perm.permission_entity_id = e.entity_id)
			`, personID, personID, personID, personID, personID, personID, personID).
		GroupBy("p.person_id").
		OrderBy(fmt.Sprintf("person_email %s", order))
	if limit != constants.MaxUint64 {
		sbuilder = sbuilder.Offset(offset).Limit(limit)
	}
	// select
	sqlr, sqla, db.err = sbuilder.ToSql()
	if db.err != nil {
		return nil, 0, db.err
	}
	if db.err = db.Select(&people, sqlr, sqla...); db.err != nil {
		return nil, 0, db.err
	}
	// count
	sqlr, sqla, db.err = cbuilder.ToSql()
	if db.err != nil {
		return nil, 0, db.err
	}
	if db.err = db.Get(&count, sqlr, sqla...); db.err != nil {
		return nil, 0, db.err
	}

	log.WithFields(log.Fields{"people": people, "count": count}).Debug("GetPeople")
	return people, count, nil
}

// GetPerson returns the person with id "id"
func (db *SQLiteDataStore) GetPerson(id int) (Person, error) {
	var (
		person Person
		sqlr   string
	)

	sqlr = "SELECT person_id, person_email FROM person WHERE person_id = ?"
	if db.err = db.Get(&person, sqlr, id); db.err != nil {
		return Person{}, db.err
	}
	return person, nil
}

// GetPersonByEmail returns the person with email "email"
func (db *SQLiteDataStore) GetPersonByEmail(email string) (Person, error) {
	var (
		person Person
		sqlr   string
	)

	sqlr = "SELECT person_id, person_email FROM person WHERE person_email = ?"
	if db.err = db.Get(&person, sqlr, email); db.err != nil {
		return Person{}, db.err
	}
	return person, nil
}

// GetPersonPermissions returns the person (with id "id") permissions
func (db *SQLiteDataStore) GetPersonPermissions(id int) ([]Permission, error) {
	var (
		ps   []Permission
		sqlr string
	)

	sqlr = `SELECT permission_id, permission_perm_name, permission_item_name, permission_entity_id 
	FROM permission
	WHERE permission_person_id = ?`
	if db.err = db.Select(&ps, sqlr, id); db.err != nil {
		return nil, db.err
	}
	log.WithFields(log.Fields{"personID": id, "ps": ps}).Debug("GetPersonPermissions")
	return ps, nil
}

// GetPersonManageEntities returns the entities the person (with id "id") if manager of
func (db *SQLiteDataStore) GetPersonManageEntities(id int) ([]Entity, error) {
	var (
		es   []Entity
		sqlr string
	)

	sqlr = `SELECT entity_id, entity_name, entity_description 
	FROM entity
	LEFT JOIN entitypeople ON entitypeople.entitypeople_entity_id = entity.entity_id
	WHERE entitypeople.entitypeople_person_id = ?`
	if db.err = db.Select(&es, sqlr, id); db.err != nil {
		return nil, db.err
	}
	log.WithFields(log.Fields{"personID": id, "es": es}).Debug("GetPersonManageEntities")
	return es, nil
}

// GetPersonEntities returns the person (with id "id") entities
func (db *SQLiteDataStore) GetPersonEntities(personID int, id int) ([]Entity, error) {
	var (
		entities []Entity
		sqlr     string
		sqla     []interface{}
	)

	sbuilder := sq.Select(`e.entity_id, 
		e.entity_id,
		e.entity_name, 
		e.entity_description`).
		From("entity AS e, person AS p, personentities as pe").
		Where("pe.personentities_person_id = ? AND e.entity_id == pe.personentities_entity_id", fmt.Sprint(id)).
		// join to filter entities personID can access to
		Join(`permission AS perm on
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "all" and perm.permission_entity_id = e.entity_id) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "all" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "all" and perm.permission_perm_name = "r" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "all" and perm.permission_entity_id = e.entity_id) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "all" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "r" and perm.permission_entity_id = -1) OR
			(perm.permission_person_id = ? and perm.permission_item_name = "entities" and perm.permission_perm_name = "r" and perm.permission_entity_id = e.entity_id)
			`, personID, personID, personID, personID, personID, personID, personID).
		GroupBy("e.entity_id")
	sqlr, sqla, db.err = sbuilder.ToSql()
	if db.err != nil {
		return nil, db.err
	}

	if db.err = db.Select(&entities, sqlr, sqla...); db.err != nil {
		return nil, db.err
	}
	return entities, nil
}

// DoesPersonBelongsTo returns true if the person (with id "id") belongs to the entities
func (db *SQLiteDataStore) DoesPersonBelongsTo(id int, entities []Entity) (bool, error) {
	var (
		sqlr  string
		count int
	)

	// extracting entities ids
	var eids []int
	for _, i := range entities {
		eids = append(eids, i.EntityID)
	}

	sqlr = `SELECT count(*) 
	FROM personentities
	WHERE personentities_person_id = ? 
	AND personentities_entity_id IN (?)`
	if db.err = db.Get(&count, sqlr, id, eids); db.err != nil {
		return false, db.err
	}
	log.WithFields(log.Fields{"personID": id, "count": count}).Debug("DoesPersonBelongsTo")
	return count > 0, nil
}

// HasPersonPermission returns true if the person with id "id" has the permission "perm" on the item "item" id "itemid"
func (db *SQLiteDataStore) HasPersonPermission(id int, perm string, item string, itemid int) (bool, error) {
	// itemid == -1 means all itemid
	// itemid == -2 means any itemid
	var (
		res     bool
		count   int
		sqlr    string
		sqlargs []interface{}
		err     error
		eids    []int
	)

	log.WithFields(log.Fields{
		"id":     id,
		"perm":   perm,
		"item":   item,
		"itemid": itemid}).Debug("HasPersonPermission")

	//
	// first: retrieving the entities of the item to be accessed
	//
	switch item {
	case "people":
		// retrieving the requested person entities
		var rpe []Entity
		if rpe, err = db.GetPersonEntities(id, itemid); err != nil {
			return false, err
		}
		// and their ids
		for _, i := range rpe {
			eids = append(eids, i.EntityID)
		}
	case "entities":
		eids = append(eids, itemid)
	}
	log.WithFields(log.Fields{"eids": eids}).Debug("HasPersonPermission")

	//
	// second: has the logged user "perm" on the "item" of the entities in "eids"
	//
	if itemid == -2 {
		// possible matchs:
		// permission_perm_name | permission_item_name
		// all | all
		// all | ?
		// ?   | all  => no sense (look at explanation in the else section)
		// ?   | ?
		sqlr = `SELECT count(*) FROM permission WHERE
		(permission_person_id = ? AND permission_perm_name = "all" AND permission_item_name = "all")  OR
		(permission_person_id = ? AND permission_perm_name = "all" AND permission_item_name = ?) OR
		(permission_person_id = ? AND permission_perm_name = ? AND permission_item_name = "all")  OR
		(permission_person_id = ? AND permission_perm_name = ? AND permission_item_name = ?)`
		if db.err = db.Get(&count, sqlr, id, id, item, id, perm, id, perm, item); db.err != nil {
			switch {
			case db.err == sql.ErrNoRows:
				return false, nil
			default:
				return false, db.err
			}
		}
	} else {
		// possible matchs:
		// permission_perm_name | permission_item_name | permission_entity_id
		// all | ?   | -1 (ex: all permissions on all entities)
		// all | ?   | ?  (ex: all permissions on entity 3)
		// ?   | all | -1 => no sense (ex: r permission on entities, store_locations...) we will put the permissions for each item
		// ?   | all | ?  => no sense (ex: r permission on entities, store_locations... with id = 3)
		// all | all | -1 => means super admin
		// all | all | ?  => no sense (ex: all permission on entities, store_locations... with id = 3)
		// ?   | ?   | -1 => (ex: r permission on all entities)
		// ?   | ?   | ?  => (ex: r permission on entity 3)
		if sqlr, sqlargs, db.err = sqlx.In(`SELECT count(*) FROM permission WHERE 
		permission_person_id = ? AND permission_item_name = "all" AND permission_perm_name = "all" OR 
		permission_person_id = ? AND permission_item_name = "all" AND permission_perm_name = ? AND permission_entity_id = -1 OR
		permission_person_id = ? AND permission_item_name = ? AND permission_perm_name = "all" AND permission_entity_id IN (?) OR
		permission_person_id = ? AND permission_item_name = ? AND permission_perm_name = "all" AND permission_entity_id = -1 OR 
		permission_person_id = ? AND permission_item_name = ? AND permission_perm_name = ? AND permission_entity_id = -1 OR
		permission_person_id = ? AND permission_item_name = ? AND permission_perm_name = ? AND permission_entity_id IN (?)
		`, id, id, perm, id, item, eids, id, item, id, item, perm, id, item, perm, eids); db.err != nil {
			return false, db.err
		}

		if db.err = db.Get(&count, sqlr, sqlargs...); db.err != nil {
			switch {
			case db.err == sql.ErrNoRows:
				return false, nil
			default:
				return false, db.err
			}
		}
	}

	log.WithFields(log.Fields{"count": count}).Debug("HasPersonPermission")
	if count == 0 {
		res = false
	} else {
		res = true
	}
	return res, nil
}

// DeletePerson deletes the person with id "id"
func (db *SQLiteDataStore) DeletePerson(id int) error {
	var (
		sqlr string
	)
	sqlr = `DELETE FROM personentities 
	WHERE personentities_person_id = ?`
	if _, db.err = db.Exec(sqlr, id); db.err != nil {
		return db.err
	}

	sqlr = `DELETE FROM entitypeople 
	WHERE entitypeople_person_id = ?`
	if _, db.err = db.Exec(sqlr, id); db.err != nil {
		return db.err
	}

	sqlr = `DELETE FROM permission 
	WHERE permission_id = ?`
	if _, db.err = db.Exec(sqlr, id); db.err != nil {
		return db.err
	}

	sqlr = `DELETE FROM person 
	WHERE person_id = ?`
	if _, db.err = db.Exec(sqlr, id); db.err != nil {
		return db.err
	}
	return nil
}

// CreatePerson creates the given person
func (db *SQLiteDataStore) CreatePerson(p Person) (error, int) {
	var (
		sqlr   string
		res    sql.Result
		lastid int64
	)

	// inserting person
	sqlr = `INSERT INTO person(person_email, person_password) VALUES (?, ?)`
	if res, db.err = db.Exec(sqlr, p.PersonEmail, p.PersonPassword); db.err != nil {
		return db.err, 0
	}

	// getting the last inserted id
	if lastid, db.err = res.LastInsertId(); db.err != nil {
		return db.err, 0
	}
	p.PersonID = int(lastid)

	// inserting permissions
	for _, per := range p.Permissions {
		sqlr = `INSERT INTO permission(permission_person_id, permission_perm_name, permission_item_name, permission_entity_id) VALUES (?, ?, ?, ?)`
		if _, db.err = db.Exec(sqlr, p.PersonID, per.PermissionPermName, per.PermissionItemName, per.PermissionEntityID); db.err != nil {
			return db.err, 0
		}
	}

	// inserting entities
	for _, e := range p.Entities {
		sqlr = `INSERT INTO personentities(personentities_person_id, personentities_entity_id) 
			VALUES (?, ?)`
		if _, db.err = db.Exec(sqlr, p.PersonID, e.EntityID); db.err != nil {
			return db.err, 0
		}
	}
	return nil, p.PersonID
}

// UpdatePerson updates the given person
func (db *SQLiteDataStore) UpdatePerson(p Person) error {
	var (
		sqlr string
	)
	// updating person
	sqlr = `UPDATE person SET person_email = ?
	WHERE person_id = ?`
	if _, db.err = db.Exec(sqlr, p.PersonEmail, p.PersonID); db.err != nil {
		return db.err
	}

	// lazily deleting former entities
	sqlr = `DELETE FROM personentities 
	WHERE personentities_person_id = ?`
	if _, db.err = db.Exec(sqlr, p.PersonID); db.err != nil {
		return db.err
	}

	// updating person entities
	for _, e := range p.Entities {
		sqlr = `INSERT INTO personentities(personentities_person_id, personentities_entity_id) 
		VALUES (?, ?)`
		if _, db.err = db.Exec(sqlr, p.PersonID, e.EntityID); db.err != nil {
			return db.err
		}
	}

	// lazily deleting former permissions
	sqlr = `DELETE FROM permission 
		WHERE permission_person_id = ?`
	if _, db.err = db.Exec(sqlr, p.PersonID); db.err != nil {
		return db.err
	}

	// updating person permissions
	for _, perm := range p.Permissions {
		sqlr = `INSERT INTO permission(permission_person_id, permission_perm_name, permission_item_name, permission_entity_id) 
		VALUES (?, ?, ?, ?)`
		if perm.PermissionPermName == "r" || perm.PermissionPermName == "w" || perm.PermissionPermName == "all" {
			if _, db.err = db.Exec(sqlr, p.PersonID, perm.PermissionPermName, perm.PermissionItemName, perm.PermissionEntityID); db.err != nil {
				return db.err
			}
		}
	}

	return nil
}

// IsPersonWithEmail returns true is the person with email "email" exists
func (db *SQLiteDataStore) IsPersonWithEmail(email string) (bool, error) {
	var (
		res   bool
		count int
		sqlr  string
	)

	sqlr = "SELECT count(*) from person WHERE person.person_email = ?"
	if db.err = db.Get(&count, sqlr, email); db.err != nil {
		return false, db.err
	}
	log.WithFields(log.Fields{"email": email, "count": count}).Debug("IsPersonWithEmail")
	if count == 0 {
		res = false
	} else {
		res = true
	}
	return res, nil
}

// IsPersonWithEmailExcept returns true is the person with email "email" exists ignoring the "except" emails
func (db *SQLiteDataStore) IsPersonWithEmailExcept(email string, except ...string) (bool, error) {
	var (
		res   bool
		count int
		sqlr  sq.SelectBuilder
		w     sq.And
	)

	w = append(w, sq.Eq{"person.person_email": email})
	for _, e := range except {
		w = append(w, sq.NotEq{"person.person_email": e})
	}

	sqlr = sq.Select("count(*)").From("person").Where(w)
	sql, args, _ := sqlr.ToSql()
	log.WithFields(log.Fields{"sql": sql, "args": args}).Debug("IsPersonWithEmailExcept")

	if db.err = db.Get(&count, sql, args...); db.err != nil {
		return false, db.err
	}
	log.WithFields(log.Fields{"email": email, "count": count}).Debug("IsPersonWithEmailExcept")
	if count == 0 {
		res = false
	} else {
		res = true
	}
	return res, nil
}
