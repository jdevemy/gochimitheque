package datastores

import (
	"database/sql"
	"strings"
	"time"

	"encoding/hex"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
	"github.com/sirupsen/logrus"
	"github.com/steambap/captcha"
	"github.com/tbellembois/gochimitheque/globals"
	. "github.com/tbellembois/gochimitheque/models"
	"github.com/tbellembois/gochimitheque/utils"
	"golang.org/x/crypto/bcrypt"
)

// ValidateCaptcha validate the text entered for the user with the given token
func (db *SQLiteDataStore) ValidateCaptcha(token string, text string) (bool, error) {

	var (
		e error
		i int
	)

	sqlr := `SELECT count(*) FROM captcha 
	WHERE captcha_token = ? AND captcha_text = ?`
	if e = db.Get(&i, sqlr, token, text); e != nil && e != sql.ErrNoRows {
		return false, e
	}
	globals.Log.WithFields(logrus.Fields{"token": token, "text": text, "i": i}).Debug("ValidateCaptcha")

	return i > 0, nil
}

// InsertCaptcha generate and stores a unique captcha with a token
// to be validated by a user, and returns the token
func (db *SQLiteDataStore) InsertCaptcha(data *captcha.Data) (string, error) {

	var (
		e     error
		uuid  []byte
		suuid string
	)

	// generating uuid for the captcha
	if uuid, e = utils.GetPasswordHash(time.Now().Format("20060102150405")); e != nil {
		return "", e
	}
	suuid = hex.EncodeToString(uuid)

	// saving
	sqlr := `INSERT INTO captcha (captcha_token, captcha_text) 
	VALUES (?, ?)`
	if _, e = db.Exec(sqlr, suuid, data.Text); e != nil {
		return "", e
	}

	return suuid, nil
}

// GetPeople returns the people matching the search criteria
// order, offset and limit are passed to the sql request
func (db *SQLiteDataStore) GetPeople(p DbselectparamPerson) ([]Person, int, error) {
	var (
		people                             []Person
		isadmin                            bool
		count                              int
		precreq, presreq, comreq, postsreq strings.Builder
		cnstmt                             *sqlx.NamedStmt
		snstmt                             *sqlx.NamedStmt
		err                                error
	)
	globals.Log.WithFields(logrus.Fields{"p": p}).Debug("GetPeople")

	// is the logged user an admin?
	if isadmin, err = db.IsPersonAdmin(p.GetLoggedPersonID()); err != nil {
		return nil, 0, err
	}

	// returning all people for admins
	// we need to handle admins
	// to see people with no entities
	precreq.WriteString("SELECT count(DISTINCT p.person_id)")
	presreq.WriteString("SELECT p.person_id, p.person_email")
	comreq.WriteString(" FROM person AS p, entity AS e")
	if p.GetEntity() != -1 {
		comreq.WriteString(" JOIN personentities ON personentities.personentities_person_id = p.person_id")
		comreq.WriteString(" JOIN entity ON personentities.personentities_entity_id = :entity")
	} else if !isadmin {
		comreq.WriteString(" JOIN personentities ON personentities.personentities_person_id = p.person_id")
		comreq.WriteString(" JOIN entity ON personentities.personentities_entity_id = e.entity_id")
	}
	if !isadmin {
		comreq.WriteString(` JOIN permission AS perm ON
		perm.person = :personid and (perm.permission_item_name in ("all", "people")) and (perm.permission_perm_name in ("all", "r", "w")) and (perm.permission_entity_id in (-1, e.entity_id))
		`)
	}
	comreq.WriteString(" WHERE p.person_email LIKE :search")
	postsreq.WriteString(" GROUP BY p.person_id")
	postsreq.WriteString(" ORDER BY " + p.GetOrderBy() + " " + p.GetOrder())

	// limit
	if p.GetLimit() != ^uint64(0) {
		postsreq.WriteString(" LIMIT :limit OFFSET :offset")
	}

	// building count and select statements
	if cnstmt, err = db.PrepareNamed(precreq.String() + comreq.String()); err != nil {
		return nil, 0, err
	}
	if snstmt, err = db.PrepareNamed(presreq.String() + comreq.String() + postsreq.String()); err != nil {
		return nil, 0, err
	}

	// building argument map
	m := map[string]interface{}{
		"entity":   p.GetEntity(),
		"search":   p.GetSearch(),
		"personid": p.GetLoggedPersonID(),
		"order":    p.GetOrder(),
		"limit":    p.GetLimit(),
		"offset":   p.GetOffset(),
	}

	// select
	if err = snstmt.Select(&people, m); err != nil {
		return nil, 0, err
	}
	// count
	if err = cnstmt.Get(&count, m); err != nil {
		return nil, 0, err
	}

	globals.Log.WithFields(logrus.Fields{"people": people, "count": count}).Debug("GetPeople")
	return people, count, nil
}

// GetPerson returns the person with id "id"
func (db *SQLiteDataStore) GetPerson(id int) (Person, error) {
	var (
		person Person
		sqlr   string
		err    error
	)

	sqlr = "SELECT person_id, person_email, person_password FROM person WHERE person_id = ?"
	if err = db.Get(&person, sqlr, id); err != nil {
		return Person{}, err
	}
	return person, nil
}

// GetPersonByEmail returns the person with email "email"
func (db *SQLiteDataStore) GetPersonByEmail(email string) (Person, error) {
	var (
		person Person
		sqlr   string
		err    error
	)

	sqlr = "SELECT person_id, person_email, person_password FROM person WHERE person_email = ?"
	if err = db.Get(&person, sqlr, email); err != nil {
		return Person{}, err
	}
	return person, nil
}

// GetPersonPermissions returns the person (with id "id") permissions
func (db *SQLiteDataStore) GetPersonPermissions(id int) ([]Permission, error) {
	var (
		ps   []Permission
		sqlr string
		err  error
	)

	sqlr = `SELECT permission_id, permission_perm_name, permission_item_name, permission_entity_id 
	FROM permission
	WHERE person = ?`
	if err = db.Select(&ps, sqlr, id); err != nil {
		return nil, err
	}
	globals.Log.WithFields(logrus.Fields{"personID": id, "ps": ps}).Debug("GetPersonPermissions")
	return ps, nil
}

// GetPersonManageEntities returns the entities the person (with id "id") if manager of
func (db *SQLiteDataStore) GetPersonManageEntities(id int) ([]Entity, error) {
	var (
		es   []Entity
		sqlr string
		err  error
	)

	sqlr = `SELECT entity_id, entity_name, entity_description 
	FROM entity
	LEFT JOIN entitypeople ON entitypeople.entitypeople_entity_id = entity.entity_id
	WHERE entitypeople.entitypeople_person_id = ?`
	if err = db.Select(&es, sqlr, id); err != nil {
		return nil, err
	}
	globals.Log.WithFields(logrus.Fields{"personID": id, "es": es}).Debug("GetPersonManageEntities")
	return es, nil
}

// GetPersonEntities returns the person (with id "id") entities
func (db *SQLiteDataStore) GetPersonEntities(LoggedPersonID int, id int) ([]Entity, error) {
	var (
		entities []Entity
		isadmin  bool
		sqlr     strings.Builder
		sstmt    *sqlx.NamedStmt
		err      error
	)
	globals.Log.WithFields(logrus.Fields{"LoggedPersonID": LoggedPersonID, "id": id}).Debug("GetPersonEntities")

	// is the logged user an admin?
	if isadmin, err = db.IsPersonAdmin(LoggedPersonID); err != nil {
		return nil, err
	}

	sqlr.WriteString("SELECT e.entity_id, e.entity_name, e.entity_description")
	sqlr.WriteString(" FROM entity AS e, person AS p, personentities as pe")
	if !isadmin {
		sqlr.WriteString(` JOIN permission AS perm ON
		(perm.person = :personid and perm.permission_item_name = "all" and perm.permission_perm_name = "all" and perm.permission_entity_id = e.entity_id) OR
		(perm.person = :personid and perm.permission_item_name = "all" and perm.permission_perm_name = "all" and perm.permission_entity_id = -1) OR
		(perm.person = :personid and perm.permission_item_name = "all" and perm.permission_perm_name = "r" and perm.permission_entity_id = -1) OR
		(perm.person = :personid and perm.permission_item_name = "entities" and perm.permission_perm_name = "all" and perm.permission_entity_id = e.entity_id) OR
		(perm.person = :personid and perm.permission_item_name = "entities" and perm.permission_perm_name = "all" and perm.permission_entity_id = -1) OR
		(perm.person = :personid and perm.permission_item_name = "entities" and perm.permission_perm_name = "r" and perm.permission_entity_id = -1) OR
		(perm.person = :personid and perm.permission_item_name = "entities" and perm.permission_perm_name = "r" and perm.permission_entity_id = e.entity_id)
		`)
	}
	sqlr.WriteString(" WHERE pe.personentities_person_id = :personid AND e.entity_id == pe.personentities_entity_id")
	sqlr.WriteString(" GROUP BY e.entity_id")
	sqlr.WriteString(" ORDER BY e.entity_name ASC")

	// building select statement
	if sstmt, err = db.PrepareNamed(sqlr.String()); err != nil {
		return nil, err
	}

	// building argument map
	m := map[string]interface{}{
		"personid": id}
	// globals.Log.Debug(sqlr)
	// globals.Log.Debug(m)

	if err = sstmt.Select(&entities, m); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	globals.Log.WithFields(logrus.Fields{"entities": entities}).Debug("GetPersonEntities")
	return entities, nil
}

// DoesPersonBelongsTo returns true if the person (with id "id") belongs to the entities
func (db *SQLiteDataStore) DoesPersonBelongsTo(id int, entities []Entity) (bool, error) {
	var (
		sqlr    string
		sqlargs []interface{}
		count   int
		err     error
	)
	globals.Log.WithFields(logrus.Fields{"id": id, "entities": entities}).Debug("DoesPersonBelongsTo")

	// extracting entities ids
	var eids []int
	for _, i := range entities {
		eids = append(eids, i.EntityID)
	}
	globals.Log.WithFields(logrus.Fields{"eids": eids}).Debug("DoesPersonBelongsTo")

	if sqlr, sqlargs, err = sqlx.In(`SELECT count(*) 
	FROM personentities 
	WHERE personentities_person_id = ? 
	AND personentities_entity_id IN (?)`, id, eids); err != nil {
		return false, err
	}
	if err = db.Get(&count, sqlr, sqlargs...); err != nil {
		return false, err
	}
	globals.Log.WithFields(logrus.Fields{"personID": id, "count": count}).Debug("DoesPersonBelongsTo")
	return count > 0, nil
}

// DeletePerson deletes the person with id "id"
func (db *SQLiteDataStore) DeletePerson(id int) error {
	var (
		sqlr  string
		err   error
		admin Person
	)
	// getting the admin
	if admin, err = db.GetPersonByEmail("admin@chimitheque.fr"); err != nil {
		return err
	}

	// updating storage ownership to admin
	sqlr = `UPDATE storage SET person = ? WHERE person = ?`
	if _, err = db.Exec(sqlr, admin.PersonID, id); err != nil {
		return err
	}

	// updating product ownership to admin
	sqlr = `UPDATE product SET person = ? WHERE person = ?`
	if _, err = db.Exec(sqlr, admin.PersonID, id); err != nil {
		return err
	}

	sqlr = `DELETE FROM personentities 
	WHERE personentities_person_id = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}

	// remove manager
	// normally not used as we can not delete a manager
	sqlr = `DELETE FROM entitypeople 
	WHERE entitypeople_person_id = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}

	sqlr = `DELETE FROM permission 
	WHERE person = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}

	sqlr = `DELETE FROM borrowing 
	WHERE borrower = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}

	sqlr = `DELETE FROM person 
	WHERE person_id = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}
	return nil
}

// CreatePerson creates the given person
func (db *SQLiteDataStore) CreatePerson(p Person) (int, error) {
	var (
		sqlr   string
		res    sql.Result
		tx     *sql.Tx
		lastid int64
		err    error
	)
	globals.Log.WithFields(logrus.Fields{"p": p}).Debug("CreatePerson")

	// beginning transaction
	if tx, err = db.Begin(); err != nil {
		return 0, err
	}

	// inserting person
	sqlr = `INSERT INTO person(person_email, person_password) VALUES (?, ?)`
	if res, err = tx.Exec(sqlr, p.PersonEmail, p.PersonPassword); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, err
	}

	// getting the last inserted id
	if lastid, err = res.LastInsertId(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, err
	}
	p.PersonID = int(lastid)

	// inserting entities
	for _, e := range p.Entities {
		sqlr = `INSERT INTO personentities(personentities_person_id, personentities_entity_id) 
			VALUES (?, ?)`
		if _, err = tx.Exec(sqlr, p.PersonID, e.EntityID); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return 0, err
			}
			return 0, err
		}
		sqlr = `INSERT INTO permission(person, permission_perm_name, permission_item_name, permission_entity_id)  
		VALUES (?, ?, ?, ?)`
		if _, err = tx.Exec(sqlr, p.PersonID, "r", "entities", e.EntityID); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return 0, err
			}
			return 0, err
		}
	}

	// inserting permissions
	if err = db.insertPermissions(p, tx); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, err
	}

	// committing changes
	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, err
	}

	return p.PersonID, nil
}

// UpdatePersonPassword updates the given person password.
func (db *SQLiteDataStore) UpdatePersonPassword(p Person) error {
	var (
		sqlr  string
		err   error
		hpass []byte
	)

	// hashing the password
	if hpass, err = bcrypt.GenerateFromPassword([]byte(p.PersonPassword), bcrypt.DefaultCost); err != nil {
		return err
	}

	// updating person
	sqlr = `UPDATE person SET person_password = ?
	WHERE person_id = ?`
	if _, err = db.Exec(sqlr, hpass, p.PersonID); err != nil {
		return err
	}
	return nil
}

// UpdatePerson updates the given person.
// The password is not updated.
func (db *SQLiteDataStore) UpdatePerson(p Person) error {
	var (
		tx   *sql.Tx
		sqlr string
		err  error
	)

	// beginning transaction
	if tx, err = db.Begin(); err != nil {
		return err
	}

	// updating person
	sqlr = `UPDATE person SET person_email = ?
	WHERE person_id = ?`
	if _, err = tx.Exec(sqlr, p.PersonEmail, p.PersonID); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
		return err
	}

	// lazily deleting former entities
	sqlr = `DELETE FROM personentities 
	WHERE personentities_person_id = ?`
	if _, err = tx.Exec(sqlr, p.PersonID); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
		return err
	}

	// lazily deleting former permissions
	sqlr = `DELETE FROM permission 
		WHERE person = ?`
	if _, err = tx.Exec(sqlr, p.PersonID); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
		return err
	}

	// updating person entities
	for _, e := range p.Entities {
		sqlr = `INSERT INTO personentities(personentities_person_id, personentities_entity_id) 
		VALUES (?, ?)`
		if _, err = tx.Exec(sqlr, p.PersonID, e.EntityID); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return err
			}
			return err
		}
		sqlr = `INSERT INTO permission(person, permission_perm_name, permission_item_name, permission_entity_id)  
		VALUES (?, ?, ?, ?)`
		if _, err = tx.Exec(sqlr, p.PersonID, "r", "entities", e.EntityID); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return err
			}
			return err
		}
	}

	// inserting permissions
	if err = db.insertPermissions(p, tx); err != nil {
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

// GetAdmins returns the administrators
func (db *SQLiteDataStore) GetAdmins() ([]Person, error) {
	var (
		people []Person
		sqlr   string
		err    error
	)
	sqlr = `SELECT person_id, person_email from person 
	JOIN permission ON 
	permission.person = person_id AND
	permission.permission_perm_name = "all" AND
	permission.permission_item_name = "all" AND
	permission_entity_id = -1 WHERE NOT
	person_email = "admin@chimitheque.fr"`
	if err = db.Select(&people, sqlr); err != nil {
		return nil, err
	}
	return people, nil
}

// IsPersonAdmin returns true is the person with id "id" is an admin
func (db *SQLiteDataStore) IsPersonAdmin(id int) (bool, error) {
	var (
		res   bool
		count int
		sqlr  string
		err   error
	)
	sqlr = `SELECT count(*) from permission WHERE 
	permission.person = ? AND
	permission.permission_perm_name = "all" AND
	permission.permission_item_name = "all" AND
	permission_entity_id = -1`
	if err = db.Get(&count, sqlr, id); err != nil {
		return false, err
	}
	globals.Log.WithFields(logrus.Fields{"id": id, "count": count}).Debug("IsPersonAdmin")
	if count == 0 {
		res = false
	} else {
		res = true
	}
	return res, nil
}

// UnsetPersonAdmin unset the person with id "id" the admin permissions
func (db *SQLiteDataStore) UnsetPersonAdmin(id int) error {
	var (
		sqlr string
		err  error
	)

	sqlr = `DELETE FROM permission WHERE person = ? AND permission_perm_name = ? AND permission_item_name = ? AND permission_entity_id = ?`
	if _, err = db.Exec(sqlr, id, "all", "all", "-1"); err != nil {
		return err
	}
	return nil
}

// SetPersonAdmin set the person with id "id" an admin
func (db *SQLiteDataStore) SetPersonAdmin(id int) error {
	var (
		isAdmin bool
		sqlr    string
		err     error
	)

	if isAdmin, err = db.IsPersonAdmin(id); err != nil {
		return err
	}
	if isAdmin {
		return nil
	}

	sqlr = `INSERT INTO permission(person, permission_perm_name, permission_item_name, permission_entity_id) 
	VALUES (?, ?, ?, ?)`
	if _, err = db.Exec(sqlr, id, "all", "all", "-1"); err != nil {
		return err
	}
	return nil
}

// IsPersonManager returns true is the person with id "id" is a manager
func (db *SQLiteDataStore) IsPersonManager(id int) (bool, error) {
	var (
		res   bool
		count int
		sqlr  string
		err   error
	)
	sqlr = "SELECT count(*) from entitypeople WHERE entitypeople.entitypeople_person_id = ?"
	if err = db.Get(&count, sqlr, id); err != nil {
		return false, err
	}
	globals.Log.WithFields(logrus.Fields{"id": id, "count": count}).Debug("IsPersonManager")
	if count == 0 {
		res = false
	} else {
		res = true
	}
	return res, nil
}

func (db *SQLiteDataStore) insertPermissions(p Person, tx *sql.Tx) error {
	var (
		sqlr string
		err  error
	)
	globals.Log.WithFields(logrus.Fields{"p.Permissions": p.Permissions}).Debug("insertPermissions")

	// inserting person permissions
	for _, perm := range p.Permissions {
		sqlr = `INSERT INTO permission(person, permission_perm_name, permission_item_name, permission_entity_id) 
		VALUES (?, ?, ?, ?)`
		if _, err = tx.Exec(sqlr, p.PersonID, perm.PermissionPermName, perm.PermissionItemName, perm.PermissionEntityID); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
	}
	return nil
}
