package datastores

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/tbellembois/gochimitheque/data"
	"github.com/tbellembois/gochimitheque/globals"
	"github.com/tbellembois/gochimitheque/models"
	. "github.com/tbellembois/gochimitheque/models"
	"github.com/tbellembois/gochimitheque/utils"
)

// SQLiteDataStore implements the Datastore interface
// to store data in SQLite3
type SQLiteDataStore struct {
	*sqlx.DB
}

var (
	regex = func(re, s string) bool {
		m, _ := regexp.MatchString(re, s)
		return m
	}
)

func init() {
	sql.Register("sqlite3_with_go_func",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("regexp", regex, true)
			},
		})
}

// GetWelcomeAnnounce returns the welcome announce
func (db *SQLiteDataStore) GetWelcomeAnnounce() (WelcomeAnnounce, error) {
	var (
		wa   WelcomeAnnounce
		sqlr string
		err  error
	)

	sqlr = `SELECT welcomeannounce.welcomeannounce_id, welcomeannounce.welcomeannounce_text
	FROM welcomeannounce LIMIT 1`
	if err = db.Get(&wa, sqlr); err != nil {
		return WelcomeAnnounce{}, err
	}

	globals.Log.WithFields(logrus.Fields{"wa": wa}).Debug("GetWelcomeAnnounce")
	return wa, nil
}

// UpdateWelcomeAnnounce updates the main page announce
func (db *SQLiteDataStore) UpdateWelcomeAnnounce(w WelcomeAnnounce) error {
	var (
		sqlr string
		tx   *sqlx.Tx
		err  error
	)

	// beginning new transaction
	if tx, err = db.Beginx(); err != nil {
		return err
	}

	// updating person
	sqlr = `UPDATE welcomeannounce SET welcomeannounce_text = ?
	WHERE welcomeannounce_id = (SELECT welcomeannounce_id FROM welcomeannounce LIMIT 1)`
	if _, err = tx.Exec(sqlr, w.WelcomeAnnounceText); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	// committing changes
	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	return nil
}

// NewSQLiteDBstore returns a database connection to the given dataSourceName
// ie. a path to the sqlite database file
func NewSQLiteDBstore(dataSourceName string) (*SQLiteDataStore, error) {
	var (
		db  *sqlx.DB
		err error
	)

	globals.Log.WithFields(logrus.Fields{"dbdriver": "sqlite3", "dataSourceName": dataSourceName}).Debug("NewDBstore")
	if db, err = sqlx.Connect("sqlite3_with_go_func", dataSourceName+"?_journal=wal&_fk=1"); err != nil {
		return &SQLiteDataStore{}, err
	}

	return &SQLiteDataStore{db}, nil
}

// ToCasbinJSONAdapter returns a JSON as a slice of bytes
// following the format: https://github.com/casbin/json-adapter#policy-json
func (db *SQLiteDataStore) ToCasbinJSONAdapter() ([]byte, error) {
	var (
		ps   []Permission
		js   []CasbinJSON
		err  error
		res  []byte
		sqlr string
	)

	sqlr = `SELECT person AS "person.person_id", permission_perm_name, permission_item_name, permission_entity_id 
	FROM permission`
	if err = db.Select(&ps, sqlr); err != nil {
		return nil, err
	}

	for _, p := range ps {
		js = append(js, models.CasbinJSON{
			PType: "p",
			V0:    strconv.Itoa(p.Person.PersonID),
			V1:    p.PermissionPermName,
			V2:    p.PermissionItemName,
			V3:    strconv.Itoa(p.PermissionEntityID),
		})
	}

	if res, err = json.Marshal(js); err != nil {
		return nil, err
	}

	return res, nil
}

// CreateDatabase creates the database tables
func (db *SQLiteDataStore) CreateDatabase() error {
	var (
		err         error
		c           int
		userVersion int
		r           *csv.Reader
		records     [][]string
	)

	// schema definition
	schema := `
	PRAGMA foreign_keys = ON;
	PRAGMA encoding = "UTF-8"; 
	PRAGMA temp_store = 2;
	PRAGMA journal_mode = WAL;
	PRAGMA temp_store = MEMORY;

	CREATE TABLE IF NOT EXISTS welcomeannounce(
		welcomeannounce_id integer PRIMARY KEY,
		welcomeannounce_text string);

	CREATE TABLE IF NOT EXISTS person(
		person_id integer PRIMARY KEY,
		person_email string NOT NULL,
		person_password string NOT NULL);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_person ON person(person_email);

	CREATE TABLE IF NOT EXISTS entity (
		entity_id integer PRIMARY KEY,
		entity_name string UNIQUE NOT NULL,
		entity_description string);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_entity ON entity(entity_name);

	CREATE TABLE IF NOT EXISTS storelocation (
		storelocation_id integer PRIMARY KEY,
		storelocation_name string NOT NULL,
		storelocation_color string,
		storelocation_canstore boolean default 0,
		storelocation_fullpath string,
		entity integer NOT NULL,
		storelocation integer,
		FOREIGN KEY(storelocation) references storelocation(storelocation_id),
		FOREIGN KEY(entity) references entity(entity_id));
	
	CREATE TABLE IF NOT EXISTS supplier (
		supplier_id integer PRIMARY KEY,
		supplier_label string NOT NULL);
	CREATE TABLE IF NOT EXISTS unit (
		unit_id integer PRIMARY KEY,
		unit_label string NOT NULL,
		unit_multiplier integer NOT NULL default 1,
		unit integer,
		FOREIGN KEY(unit) references unit(unit_id));
	CREATE TABLE IF NOT EXISTS storage (
		storage_id integer PRIMARY KEY,
		storage_creationdate datetime NOT NULL,
		storage_modificationdate datetime NOT NULL,
		storage_entrydate datetime,
		storage_exitdate datetime,
		storage_openingdate datetime,
		storage_expirationdate datetime,
		storage_quantity float,
		storage_barecode string,
		storage_comment string,
		storage_reference string,
		storage_batchnumber string,
		storage_todestroy boolean default 0,
		storage_archive boolean default 0,
		storage_qrcode blob,
		person integer NOT NULL,
		product integer NOT NULL,
		storelocation integer NOT NULL,
		unit integer,
		supplier integer,
		storage integer,
		FOREIGN KEY(storage) references storage(storage_id),
		FOREIGN KEY(unit) references unit(unit_id),
		FOREIGN KEY(supplier) references supplier(supplier_id),
		FOREIGN KEY(person) references person(person_id),
		FOREIGN KEY(product) references product(product_id),
		FOREIGN KEY(storelocation) references storelocation(storelocation_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_product ON storage(storage_id, product);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_storelocation ON storage(storage_id, storelocation);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_storelocation_product ON storage(storage_id, storelocation, product);

	CREATE TABLE IF NOT EXISTS borrowing (
		borrowing_id integer PRIMARY KEY,
		borrowing_comment string,
		person integer NOT NULL,
		borrower integer NOT NULL,
		storage integer NOT NULL UNIQUE,
		FOREIGN KEY(person) references person(person_id),
		FOREIGN KEY(storage) references storage(storage_id),
		FOREIGN KEY(borrower) references person(person_id)
	);

	-- person permissions
	CREATE TABLE IF NOT EXISTS permission (
		permission_id integer PRIMARY KEY,
		person integer NOT NULL,
		permission_perm_name string NOT NULL,
		permission_item_name string NOT NULL,
		permission_entity_id integer,
		FOREIGN KEY(person) references person(person_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_permission ON permission(person, permission_item_name, permission_perm_name, permission_entity_id);

	-- entities people belongs to
	CREATE TABLE IF NOT EXISTS personentities (
		personentities_person_id integer NOT NULL,
		personentities_entity_id integer NOT NULL,
		PRIMARY KEY(personentities_person_id, personentities_entity_id),
		FOREIGN KEY(personentities_person_id) references person(person_id),
		FOREIGN KEY(personentities_entity_id) references entity(entity_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_personentities ON personentities(personentities_person_id, personentities_entity_id);

	-- entities managers	
	CREATE TABLE IF NOT EXISTS entitypeople (
		entitypeople_entity_id integer NOT NULL,
		entitypeople_person_id integer NOT NULL,
		PRIMARY KEY(entitypeople_entity_id, entitypeople_person_id),
		FOREIGN KEY(entitypeople_person_id) references person(person_id),
		FOREIGN KEY(entitypeople_entity_id) references entity(entity_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_entitypeople ON entitypeople(entitypeople_entity_id, entitypeople_person_id);

	-- products symbols
	CREATE TABLE IF NOT EXISTS symbol (
		symbol_id integer PRIMARY KEY,
		symbol_label string NOT NULL,
		symbol_image string);

	-- products names
	CREATE TABLE IF NOT EXISTS name (
		name_id integer PRIMARY KEY,
		name_label string NOT NULL UNIQUE);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_name ON name(name_label);

	-- products cas numbers
	CREATE TABLE IF NOT EXISTS casnumber (
		casnumber_id integer PRIMARY KEY,
		casnumber_label string NOT NULL UNIQUE,
		casnumber_cmr string);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_casnumber ON casnumber(casnumber_label);

	-- products ce numbers
	CREATE TABLE IF NOT EXISTS cenumber (
		cenumber_id integer PRIMARY KEY,
		cenumber_label string NOT NULL UNIQUE);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_cenumber ON cenumber(cenumber_label);

	-- products empirical formulas
	CREATE TABLE IF NOT EXISTS empiricalformula (
		empiricalformula_id integer PRIMARY KEY,
		empiricalformula_label string NOT NULL UNIQUE);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_empiricalformula ON empiricalformula(empiricalformula_label);

	-- products linear formulas
	CREATE TABLE IF NOT EXISTS linearformula (
		linearformula_id integer PRIMARY KEY,
		linearformula_label string NOT NULL UNIQUE);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_linearformula ON linearformula(linearformula_label);

	-- products physical states
	CREATE TABLE IF NOT EXISTS physicalstate (
		physicalstate_id integer PRIMARY KEY,
		physicalstate_label string NOT NULL UNIQUE);

	-- products signal words
	CREATE TABLE IF NOT EXISTS signalword (
		signalword_id integer PRIMARY KEY,
		signalword_label string NOT NULL UNIQUE);

	-- products classes of compound
	CREATE TABLE IF NOT EXISTS classofcompound (
		classofcompound_id integer PRIMARY KEY,
		classofcompound_label string NOT NULL UNIQUE);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_classofcompound ON classofcompound(classofcompound_label);

	-- products hazard statements
	CREATE TABLE IF NOT EXISTS hazardstatement (
		hazardstatement_id integer PRIMARY KEY,
		hazardstatement_label string NOT NULL,
		hazardstatement_reference string NOT NULL);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_hazardstatement ON hazardstatement(hazardstatement_reference);

	-- products precautionary statements
	CREATE TABLE IF NOT EXISTS precautionarystatement (
		precautionarystatement_id integer PRIMARY KEY,
		precautionarystatement_label string NOT NULL,
		precautionarystatement_reference string NOT NULL);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_precautionarystatement ON precautionarystatement(precautionarystatement_reference);

	-- products
	CREATE TABLE IF NOT EXISTS product (
		product_id integer PRIMARY KEY,
		product_specificity string,
		product_msds string,
		product_restricted boolean default 0,
		product_radioactive boolean default 0,
		product_threedformula string,
		product_molformula blob,
		product_disposalcomment string,
		product_remark string,
		product_qrcode string,
		casnumber integer,
		cenumber integer,
		person integer NOT NULL,
		empiricalformula integer NOT NULL,
		linearformula integer,
		physicalstate integer,
		signalword integer,
		name integer NOT NULL,
		FOREIGN KEY(casnumber) references casnumber(casnumber_id),
		FOREIGN KEY(cenumber) references cenumber(cenumber_id),
		FOREIGN KEY(person) references person(person_id),
		FOREIGN KEY(empiricalformula) references empiricalformula(empiricalformula_id),
		FOREIGN KEY(linearformula) references linearformula(linearformula_id),
		FOREIGN KEY(physicalstate) references physicalstate(physicalstate_id),
		FOREIGN KEY(signalword) references signalword(signalword_id),
		FOREIGN KEY(name) references name(name_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_product_casnumber ON product(product_id, casnumber);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_product_cenumber ON product(product_id, cenumber);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_product_empiricalformula ON product(product_id, empiricalformula);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_product_name ON product(product_id, name);

	CREATE TABLE IF NOT EXISTS productclassofcompound (
		productclassofcompound_product_id integer NOT NULL,
		productclassofcompound_classofcompound_id integer NOT NULL,
		PRIMARY KEY(productclassofcompound_product_id, productclassofcompound_classofcompound_id),
		FOREIGN KEY(productclassofcompound_product_id) references product(product_id),
		FOREIGN KEY(productclassofcompound_classofcompound_id) references classofcompound(classofcompound_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_productclassofcompound ON productclassofcompound(productclassofcompound_product_id, productclassofcompound_classofcompound_id);

	CREATE TABLE IF NOT EXISTS productsymbols (
		productsymbols_product_id integer NOT NULL,
		productsymbols_symbol_id integer NOT NULL,
		PRIMARY KEY(productsymbols_product_id, productsymbols_symbol_id),
		FOREIGN KEY(productsymbols_product_id) references product(product_id),
		FOREIGN KEY(productsymbols_symbol_id) references symbol(symbol_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_productsymbols ON productsymbols(productsymbols_product_id, productsymbols_symbol_id);

	CREATE TABLE IF NOT EXISTS productsynonyms (
		productsynonyms_product_id integer NOT NULL,
		productsynonyms_name_id integer NOT NULL,
		PRIMARY KEY(productsynonyms_product_id, productsynonyms_name_id),
		FOREIGN KEY(productsynonyms_product_id) references product(product_id),
		FOREIGN KEY(productsynonyms_name_id) references name(name_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_productsynonyms ON productsynonyms(productsynonyms_product_id, productsynonyms_name_id);

	CREATE TABLE IF NOT EXISTS producthazardstatements (
		producthazardstatements_product_id integer NOT NULL,
		producthazardstatements_hazardstatement_id integer NOT NULL,
		PRIMARY KEY(producthazardstatements_product_id, producthazardstatements_hazardstatement_id),
		FOREIGN KEY(producthazardstatements_product_id) references product(product_id),
		FOREIGN KEY(producthazardstatements_hazardstatement_id) references hazardstatement(hazardstatement_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_producthazardstatements ON producthazardstatements(producthazardstatements_product_id, producthazardstatements_hazardstatement_id);

	CREATE TABLE IF NOT EXISTS productprecautionarystatements (
		productprecautionarystatements_product_id integer NOT NULL,
		productprecautionarystatements_precautionarystatement_id integer NOT NULL,
		PRIMARY KEY(productprecautionarystatements_product_id, productprecautionarystatements_precautionarystatement_id),
		FOREIGN KEY(productprecautionarystatements_product_id) references product(product_id),
		FOREIGN KEY(productprecautionarystatements_precautionarystatement_id) references precautionarystatement(precautionarystatement_id));
	CREATE UNIQUE INDEX IF NOT EXISTS idx_productprecautionarystatements ON productprecautionarystatements(productprecautionarystatements_product_id, productprecautionarystatements_precautionarystatement_id);

	CREATE TABLE IF NOT EXISTS bookmark (
		bookmark_id integer PRIMARY KEY,
		person integer NOT NULL,
		product integer NOT NULL,
		FOREIGN KEY(person) references person(person_id),
		FOREIGN KEY(product) references product(product_id));
		
	CREATE TABLE IF NOT EXISTS captcha (
		captcha_id integer PRIMARY KEY,
		captcha_token string NOT NULL,
		captcha_text string NOT NULL);
	`

	// values definition
	inssymbol := `INSERT INTO symbol (symbol_label, symbol_image) VALUES 
	("SGH01", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAInSURBVFiFzdi9b45RGMfxz11KNalS0napgVaE0HhLxEtpVEIQxKAbiUhMJCoisTyriYTBLCZGsRCrP8JkkLB4iVlyDL1EU9refZ7raZ3kJPfrub75/a7zWpVSpJSqaoBSGintlVJarzQKJWojo81sqDS4XKUSlcu3LwkuFyoRLh8qCa49UAlw6VDoRUercOlK4T7WtapcNlQHHmfYmgm1CVO4MOPZBow31V4SVDde4i22xrMeXEfVVK5myB4gh/AQVwPqSljbEe/XYVfd9lOgIvAt3AnAS1iBczgV168wVTdOClSAPcMwzmIg4EbRP+u7behZKF6r9q3BTTzFC1wLO49iD/owHioex2nswGpsnC9uU1BYhUE8R8EH3As1DuIYtmAnDsT9SZwPJScxMp8o9RKRtQHSFUk8jBHcxpPIr95QqC+svIxHGKiVDrM4VqpRSik/qqoaxTecwSe8CUWO4Dve4W6o9xFf8Bl9VVV1RgfoDLXfl1J+LhR0bp+nVRjGZoxhLw7jRNhzIwAKXmMCD/AVDVxsRq3ayY/1GEK/6RF+u+k5cTAUGJoxVk1ionaPnjf568HtD6h9GJunY3RjN7qahfobrEYP9Xv0brUuaoCt+VO7oeYGaydcS5N4u+BSlj3ZcKkLxSy4tiytW4Vr62ak2SBLsn1bbLAl3fDWDbosRwQLBV/WQ5W5IP6LY6h/w6VA5YAl2jez1lrBLlhKaaiqP9cJ5Rf+De5Q3HyidwAAAABJRU5ErkJggg=="),
	("SGH02", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAIvSURBVFiFzdgxbI5BGMDx36uNJsJApFtFIh2QEIkBtYiYJFKsrDaJqYOofhKRMFhsFhNNMBgkTAaD0KUiBomN1EpYBHWGnvi8vu9r7/2eti65vHfv3T3P/57nuXvfuyqlJCRVVQuk1AqRl1LqP9NKpJxbETKjocLgYi0VaLl49wXBxUIFwsVDBcGFQWEAg1FwYZbCGMajLBfmPkzgUZRbw2IKFzGPrRFw/bpvD/bn8jUkXM719f3A9eu+k3iXA/92Bnub2yYx1NgDfbrvXIYZx8dcThjBExxvPOmGltqLIzmuEt63QSVczc+z/2whSw2ThpbajS+4UgOq59O4gYFSuGaByWb8zKvwN8RXXKiBPc7PLaWx3ARqY37O1CBe5/cvO1huVy+ZnfSX7y9MYxRTNeX32lZj+/sXWNfVnV3g1tT/aJeQ5vAGp3L9eXbjTFv7NzzM9VncSSnNF2lp4MqjNYvcxwEcy+0HcQg32/q8Kndl+YrcgM9Z4YdsrZ21PtvxHT9yv1vNgr8cbiIrnMUmbKu177PwVZjLgKPNt4sCOKzF0ww32aF9CA+yxSZKoTqDlVnucI6lMxhpg76OuxhrKr8oIENyXx/xxQKTE/hUkIdLJ1tlRd3TwtF/KtcuSalVVdUwdvQe+Fd6ljhfl9NzRKT5I8cvq/B+xi3vzFfk+FaqbEUPvEtVuipXBIspX9VLlW4Q/8U1VGe4EKgYsED3tefBgt271y7dUlV/ygHpF8bRglXiwx7BAAAAAElFTkSuQmCC"),
	("SGH03", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAJhSURBVFiFzdjNq41RFMfxz/FSXK4iUV4GUjJRMpG8lQnJUF0mTAy8xkApimOEMDA2oSgD5R9gcE0obxO6JkooBroJhRttg7Nu94lzzz37OdvLrt16zrPXs9d3/9Ze+5zzNFJKirRGowlSahaZL6XUe6eZSNGbJeYsDVUMrqxSBZUrn75CcMWgsARTSsEVUwq7sLWUcmXS1wK7iOvd+pcDa5++qVgU1/fxKq4n9wrXa/qW4Bb6MIKE2diPmb3A1VYq7MqAuRQ2YTdeY6CXtNZVaj4uYG0FaLR/D3sc0+vC1d3okwJgsAI0iB+Vz5dxe1TdXLg6UHPCvg2AT2E34VobBaflzD8+2AQPRYqu4kUEPh1KzcKOuPck7CMcQF92nOyVtCquqsg8PI5C2IyHWBFjn8NuzM5Mdu7ZGcGO4k1U5EgF9CNO4QuG4t6x7ALLrhY2RLB9uBMAJ7Ea63A+CuMVlobvidzqzz9fmFtR5jvWtPHZHj4Xww5MNO+vHJNktpTSezxAP26klO618bkZah4JRe/mxslOZSiyLZQ43MHnTPicy1Wr1uavBH6Hsx3Gr+ADZudC1TouKoFv4CX624wtwDBO1oH6HSwDDsvxTetrZ2Hl/jKtg3UYs+pAtQfLg1uldcqPhH2qVanPsL4u1Phg3ayIPdiLg3hu7IAdwqEY24vFtRZdew/wtQLTqW+ptYc7gnWYLPbS8i76jFyo7sBqTFri+T86eS/P/dmV/5W/b7nB/uof3m6D/pNXBBMF/6cvVcaD+C9eQ7WHKwJVBqxg+qp9SvYvy3YtpaZGY+y6QPsJlPiFVobY9AkAAAAASUVORK5CYII="),
	("SGH04", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAFtSURBVFiFzdixLgRBGADgb4UoBYXKA3gAHYU3EKWH0FKuAk8gGm+hvU4oJAqlaDVqiUSCUbjEJe5ys7v/7NnkT66Ynf/bmZ3bf6ZKKQm5qqoGKdUh/aWUugd1Ig2jjugzGhWGix2pwJGLn74gXCwqEBePCsKVQQXgyqE63lcW1eH+8qiW/fSDatFff6iG/faLatB//6jMPKEoLGEX53jEHTaxj0sMcvN1QmEBWzjGLT6QxsQX7nGUO3KNUdjAAa7wOgEyGjdYazqt+auEQzxnQEbjBdtt3rn5nCq3qqp1nGJuStM3XGMwjIc0fKrGV9YKYQXv/o7Ip58X/AQ7WIxaofnLl7Mh5gkX2MNyqb+NrEYjuNXOkMx8jRr3hRoP6wPX6pNUGtfpI14KF1L2RONCC8UoXJHSuiuu6GakbZJetm9Nk/W64c1NOpMjgmnJZ3qoMgnxL46hxuNCUDGwwOkbjawKNqParFXV7++A6xtDLLIHRMAuWAAAAABJRU5ErkJggg=="),
	("SGH05", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAI9SURBVFiFzdhPiI1RGMfxz2FmgYVo1CyYhVJISuPPDmFGNwtW2EzKn4TEELKaa2NNdiyUlWxtUGYptiytrRSNIk3GsbinceO+7vvOPe8dp56677k9v+d7nnPOczonxBhlaSE0QYzNLHoxxt6NZiQma+bQzA2VDS5vpjJmLv/0ZYLLC5URLj9UJrh6oDLA1QfVo1+9UD341w+1QJ2exLAG+3AKq9DAGUxgJ1YsFG6gy9k3lb5uibEZQhjHNnxFxEe8w1ucwxfcxxw24GAIYTkCBmOMTSFIulNCKDxbO4N1gEq/P6SsfMdw6hvFDB5hGS5iLYZSRr/hE6bRAikDV2X6sC4B7MZQ13XSAlufII5iBGOllknFNTWGq6W3PJO4ngYygRu4WSoJJTO1N62TXWnqtmK0BNgR3E4DmsQdjJRJRtlMNXAX53EBz7G6BNgAjmEH9hT6dIhfagtjCV5gC67hUgmoA3iNl3iDe7iSQAe7wRWWixBCAyfauj5jo9bOPBRCeNz239kY40yb72lsx5MEth9PY4zvQwgr8aMo7nwrTCWXtWpVGRv+I1vH8SxlaRxLsblKIQ9J6K/aFXiITV1H1mrTMcbZ9o4QwmGc1CrGr/BTa83N4kGMca5T3PmattAjI4uVKhf9hqtUYPsFV6YS9OJcF9S/weqAq6CXVSynTi2iOfxrFe/Fr96R9+X6VjVYXy+8ZYMuyhNBt+CL+qhSBPFfPEN1hssClQcs4/S1W/GFt0r7fVcsvMBWbb8AgnCJLinP5ycAAAAASUVORK5CYII="),
	("SGH06", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAK6SURBVFiFzdhPiFdVFMDxzxsnpTIFNQpJZjEoVEwQyoRYRP82JUQySAYuZITEDBeCuvPnInXhzGAaRTMtDNyMBElUi9m4ECVGI6FVMOuilUG4COO0eBfmj2/m997vd0e7cOG9y7vnfN8555577i0iQpZWFC0Q0coiLyK677SCSL2VQ2ZuqGxweS2V0XL53ZcJLi9URrj8UJnglgcqA1zXUHgc7+EsvsQxbOwWrluoAXyDHXgkjfXhC7zbDVw3UIO4gFWVgjmIjzqF6xTqhWSVNxcVzAq0cKgTuI4CF6cxgr4lhTOG850siMZQSeEr+B29NcBONZVfDba0pXrmPH+Ll7EX7+OxNP56GtuBy1iXxot5P9Lu5xtADWICn6AHn2M3Av9gc/puOo19jXNp7FlM4kJtfQ3cdx39eAPncS4p3IPXUrD34xkcwUZ8hW24hJXJtUO1Flhtn/M9Pk6JdBc+wx8pLfRgP+5gLV7ET7iRIA+keXtxpk74FFEu6RNl2eikRSrQoijeSop/w3Cyzq94CbeTpQaSu99JMXUN2zGOX7ATP0TE3xUK5nH0VEFUtYiYiojpiPgrIkZxFE/jJp7HFA4rF8R3CXImIvZFxPWIuBsRk5VQiyhsnpVL1w0pk+wVPKeMvQ8T7EVlDI5hSzt51a4sFd1nyoUuLYriOJ5Srrgn8LPSnffQi1X4E5twFa9iBquxRunaf/FjREzNEVytt9YKKS00jrH0vi8J2zDnm0ct2AmU8TehjL+VuIUPmqWL9nCFsqxZj5G27pmdt1a52W/FcGcJtj3cIEbxZF2wNO9tfIrVtdNTk4DM0rvaxJcLLkvZkxsua6GYC25ZSutu4Zb1MNKpkgdyfGuq7IEeeOsqfShXBO2UP9RLlcUg/hfXUNVwWaDygGV039zeW10+NmwRLUUx+5yh/QdzLVcJBJ5ddQAAAABJRU5ErkJggg=="),
	("SGH07", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAF/SURBVFiFzdghT8NAGMbx/y1g0HMLQWDxJDgCkk+AQhE8BlkDGL4JwczyAXBkCY4vQAhBgiI7xJpwK1fW9nm6ccklNe/7/u7eW9c2xBixjBAKAGIsLPlijPqEIkIsZ+HI6UbZcN6dMu6cv30mnBdlxPlRJlw/KAOuP5QYJycHBsARMAYmwCUwVHHyioELIFbmvbpzchuAmwzsSW2rfDaA0wzsTs4rrwz2M7BruRPyWYBRBnYin10FVcIC8FGB7TWJ/fPXrqAS3KQCGzaOr6kro0rYbYJ6bxufq+/5w4WrBPbQJUfVMZAfgWfjuea6+zC1cht4Y7Zjx55WGg5/idsEDhyoeZiIA7aAXQfqN6wjDjgHvoApi+76Det0CsrAXpi/j+0oqHpYSxzwmKCmwEjNK213AjsEXoFP4MyyWPUsJLh1YMOBagbrkNQR32tyJa7flS/l9a1tsaW+8DYtupJPBIuKr/SjSh3iX3yGyuMsKA/M2L50rpmeNgtC+Lk2jG/Rx4o589viKwAAAABJRU5ErkJggg=="),
	("SGH08", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAKJSURBVFiFzdixix1FGADw38QDUbQQqyNewiEYCxNN4ECjkuIaFYLNQQpTCBLkzD8gJsVLOqPGJiBYeGB1SeFhYXEhmCIpTgIpRDAEosWhckXU6AknCUyKN49bn+/tze6OnAsDuzA789v5vvnevg0xRkWOEHogxl6R8WKM3Ru9SEytV2LM0qhiuLIrVXDlyoevEK4sqiCuGApTeBo7SuA6o7AX3yKmdh3PdsV1RU3hzwpq0NbxZBdc1/AtjEAN2lddcq41KsG+qYH9jYfb4lqjEuxWDSxissl4ebCMQbBSg/qp7bjjYbk3c7oGttjpoduiEuz5GthbndKkLaqCu5Qgd/FdOv8FD3XaWF1QCfZawqzhWDo/mnv/2FLUEfUA3kuYL/FyOn87GzZm/rbhm8JH+LmSU6t4pHL9A87gmTa4Hf9+px19hBAmQgjzIYQVnMIGHqt0+QKvVq6ncRAzIYSlEMLZEMKu3PmyQ4kTldWI+lX/MD7GZ5jHJH7ESbyJs/qbYnDP79idF8rc7Tu6mK7hHexJfQLewE7jfxWO5Cd/zvblypiJ/sBzQ32vjun7T1hWudjqCdiH20OTrOJz7MJTqd8reAk3RqCu4fGsCDWqLezHr2mSdcxgFss4nkK4iPcxh8v6myTiazyandONCx8H8Ck+wM3KapxP+Ta43tDfqXM4hweb1MzGha+CuzMUpoWU+MPh+yR3g+XD6nEvpMQfAJaxNIT6EKEpKg9Wj3vR5jv/b/r1a4B6t81KNYPV4w7hL3yfcu+e6ivPf/pnZGvcLC7iAl7vimoOq8dN44kSqHawnEm35RPBVpNv60eVcYj/xWeo0bgiqDKwguGrtgkljhh7Qtg8L3DcB497IINNg8B2AAAAAElFTkSuQmCC"),
	("SGH09", "image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACYAAAAmCAYAAACoPemuAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAN1wAADdcBQiibeAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAI8SURBVFiFzdi7a1VBEMDhb0EjYhFfaCEICopBi4haWUiqqGgQW1OKaBpNIQEVvREi2lhaWCiC/4CSImgv2IgQELSSdKKlMUKEtTgr3jxuPI+9iQMD5+y9u/NjdmbO7IYYoywSQgvE2MqyXoyxudKKxKStHGvmhsoGl9dTGT2Xf/syweWFygiXHyoTXHegMsDVgprjLk5hS4wRenLD1fYUhjGans9jF/rRlwOuMhS2Yz924F0CO4EBjOTyXK0YwUVcwAcEHMJr9OaKudqLYBA/cQx7cT+NB1xrCld5cvLOGA7gqSIRricv7sED9Df1XB1PbcVOXMZnzOA7pvEcg3UTqjNY1QDlKmLSsRX+14NNlXamLlQyeDJBfenw+xmcxQ3cxunSCVYpINmYatUfHUlg3xaN9+MmjqR5uxPkaNltrZbChcFYUi8tmjuOqbIxt25pT7uifMThtvcBPMS84kvwKY2fizE+hhDCBkXGzuBZaUsN4qtPUVQj3uOJIhmGFSXlOO4loEmsr5KhtYIfvRjCUTxS1K8reKnIwAkLt/UHhqqUjUblIkFuxi18TRCv8KsNah5v8UbqRqqVi5JwOKhI/ym8wKzlg3+xzmGyrJ2QjC2U4ox4J72NS2fFEMI27CsdwEtlNsY43Wn9BVIlILNoo494t+CytD254bI2irngutJaN4Xr6mGkrpFVOb5VNbaqB96yRtfkiuBfxtf0UqUTxH9xDbU8XBaoPGAZt69dq3awy0uMLSH8fc4gvwFyuYuihNiCxwAAAABJRU5ErkJggg==");`
	inssignalword := `INSERT INTO signalword (signalword_label) VALUES ("danger"), ("warning")`
	inswelcomeannounce := `INSERT INTO welcomeannounce (welcomeannounce_text) VALUES ("")`

	// tables creation
	globals.Log.Info("  creating sqlite tables")
	if _, err = db.Exec(schema); err != nil {
		return err
	}

	// shema migration
	if err = db.Get(&userVersion, `PRAGMA user_version`); err != nil {
		return err
	}
	globals.Log.Info(fmt.Sprintf("  user_version:%d", userVersion))

	switch userVersion {
	case 0:
		migrationOne := `BEGIN TRANSACTION;
		ALTER TABLE hazardstatement ADD hazardstatement_cmr string;
		
		UPDATE hazardstatement SET hazardstatement_cmr='M1' WHERE hazardstatement_reference='H340';
		UPDATE hazardstatement SET hazardstatement_cmr='M2' WHERE hazardstatement_reference='H341';
		UPDATE hazardstatement SET hazardstatement_cmr='C1' WHERE hazardstatement_reference='H350';
		UPDATE hazardstatement SET hazardstatement_cmr='C1' WHERE hazardstatement_reference='H350i';
		UPDATE hazardstatement SET hazardstatement_cmr='C2' WHERE hazardstatement_reference='H351';
		UPDATE hazardstatement SET hazardstatement_cmr='R1' WHERE hazardstatement_reference='H360';
		UPDATE hazardstatement SET hazardstatement_cmr='R1' WHERE hazardstatement_reference='H360F';
		UPDATE hazardstatement SET hazardstatement_cmr='R1' WHERE hazardstatement_reference='H360D';
		UPDATE hazardstatement SET hazardstatement_cmr='R1' WHERE hazardstatement_reference='H360Fd';
		UPDATE hazardstatement SET hazardstatement_cmr='R1' WHERE hazardstatement_reference='H360Df';
		UPDATE hazardstatement SET hazardstatement_cmr='R1' WHERE hazardstatement_reference='H360FD';
		UPDATE hazardstatement SET hazardstatement_cmr='R2' WHERE hazardstatement_reference='H361';
		UPDATE hazardstatement SET hazardstatement_cmr='R2' WHERE hazardstatement_reference='H361f';
		UPDATE hazardstatement SET hazardstatement_cmr='R2' WHERE hazardstatement_reference='H361d';
		UPDATE hazardstatement SET hazardstatement_cmr='R2' WHERE hazardstatement_reference='H361fd';
		UPDATE hazardstatement SET hazardstatement_cmr='L' WHERE hazardstatement_reference='H362';

		PRAGMA user_version=1;
		COMMIT;
		`
		globals.Log.Info("  migrating to user_version 1")
		if _, err = db.Exec(migrationOne); err != nil {
			return err
		}

		globals.Log.Info("Please restart the application to perform the next migration.")
		os.Exit(0)
	case 1:
		migrationTwo := `BEGIN TRANSACTION;
		
		DELETE FROM permission WHERE permission_item_name='storelocations';
		DELETE FROM permission WHERE permission_id IN (SELECT p1.permission_id FROM permission p1 INNER JOIN permission p2 WHERE p1.person=p2.person AND p1.permission_perm_name="r" AND p2.permission_perm_name="w" AND p1.permission_item_name=p2.permission_item_name AND p1.permission_entity_id=p2.permission_entity_id);

		PRAGMA user_version=2;
		COMMIT;
		`
		globals.Log.Info("  migrating to user_version 2")
		if _, err = db.Exec(migrationTwo); err != nil {
			return err
		}

		globals.Log.Info("Please restart the application to perform the next migration.")
		os.Exit(0)
	case 2:
		migrationThree := `PRAGMA foreign_keys=off;
		BEGIN TRANSACTION;
		
		CREATE TABLE IF NOT EXISTS new_unit (
			unit_id integer PRIMARY KEY,
			unit_label string UNIQUE NOT NULL,
			unit_multiplier integer NOT NULL default 1,
			unit_type string,
			unit integer,
			FOREIGN KEY(unit) references unit(unit_id));

		INSERT into new_unit (
			unit_id,
			unit_label,
			unit_multiplier,
			unit
		)
		SELECT unit_id,
			unit_label,
			unit_multiplier,
			unit
		FROM unit;

		DROP table unit;
		ALTER TABLE new_unit RENAME TO unit; 

		INSERT OR IGNORE INTO unit (unit_label) VALUES ("L"), ("mL"), ("µL");
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="L"), unit_multiplier=0.001 WHERE unit_label="mL";
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="L"), unit_multiplier=0.000001 WHERE unit_label="µL";
		UPDATE unit SET unit_multiplier=1 WHERE unit_label="L";

		INSERT OR IGNORE INTO unit (unit_label) VALUES ("kg"), ("g"), ("mg"), ("µg");
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="g"), unit_multiplier=1000 WHERE unit_label="kg";
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="g"), unit_multiplier=0.001 WHERE unit_label="mg";
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="g"), unit_multiplier=0.000001 WHERE unit_label="µg";
		UPDATE unit SET unit_multiplier=1 WHERE unit_label="g";

		INSERT OR IGNORE INTO unit (unit_label) VALUES ("m"), ("dm"), ("cm");
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="m"), unit_multiplier=10 WHERE unit_label="dm";
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="m"), unit_multiplier=100 WHERE unit_label="cm";
		UPDATE unit SET unit_multiplier=1 WHERE unit_label="m";

		INSERT OR IGNORE INTO unit (unit_label) VALUES ("°K"), ("°F"), ("°C");
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="°K") WHERE unit_label="°F";
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="°K") WHERE unit_label="°C";

		INSERT OR IGNORE INTO unit (unit_label) VALUES ("nM"), ("µM"), ("mM");
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="mM") WHERE unit_label="µM";
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="mM") WHERE unit_label="mM";

		INSERT OR IGNORE INTO unit (unit_label) VALUES ("ng/L"), ("µg/L"), ("mg/L"), ("g/L");
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="g/L") WHERE unit_label="ng/L";
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="g/L") WHERE unit_label="µg/L";
		UPDATE unit SET unit=(SELECT unit_id FROM unit WHERE unit_label="g/L") WHERE unit_label="mg/L";

		UPDATE unit SET unit_type="quantity" WHERE unit_label="L";
		UPDATE unit SET unit_type="quantity" WHERE unit=(SELECT unit_id FROM unit WHERE unit_label="L");
		UPDATE unit SET unit_type="quantity" WHERE unit_label="g";
		UPDATE unit SET unit_type="quantity" WHERE unit=(SELECT unit_id FROM unit WHERE unit_label="g");
		UPDATE unit SET unit_type="quantity" WHERE unit_label="m";
		UPDATE unit SET unit_type="quantity" WHERE unit=(SELECT unit_id FROM unit WHERE unit_label="m");
		UPDATE unit SET unit_type="temperature" WHERE unit_label="°K";
		UPDATE unit SET unit_type="temperature" WHERE unit=(SELECT unit_id FROM unit WHERE unit_label="°K");
		UPDATE unit SET unit_type="concentration" WHERE unit_type IS NULL;

		CREATE TABLE IF NOT EXISTS tag (
			tag_id integer PRIMARY KEY,
			tag_label string NOT NULL);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_label ON tag(tag_label);

		CREATE TABLE IF NOT EXISTS category (
			category_id integer PRIMARY KEY,
			category_label string NOT NULL);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_category_label ON category(category_label);

		CREATE TABLE IF NOT EXISTS producer (
			producer_id integer PRIMARY KEY,
			producer_label string NOT NULL);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_producer_label ON producer(producer_label);

		CREATE TABLE IF NOT EXISTS producerref (
			producerref_id integer PRIMARY KEY,
			producerref_label string NOT NULL,
			producer integer,
			FOREIGN KEY(producer) references producer(producer_id));
		CREATE UNIQUE INDEX IF NOT EXISTS idx_producerref_label ON producerref(producerref_label);

		CREATE TABLE IF NOT EXISTS supplierref (
			supplierref_id integer PRIMARY KEY,
			supplierref_label string NOT NULL,
			supplier integer,
			FOREIGN KEY(supplier) references supplier(supplier_id));
		CREATE UNIQUE INDEX IF NOT EXISTS idx_supplierref_label ON supplierref(supplierref_label);

		CREATE TABLE IF NOT EXISTS productsupplierrefs (
			productsupplierrefs_product_id integer NOT NULL,
			productsupplierrefs_supplierref_id integer NOT NULL,
			PRIMARY KEY(productsupplierrefs_product_id, productsupplierrefs_supplierref_id),
			FOREIGN KEY(productsupplierrefs_product_id) references product(product_id),
			FOREIGN KEY(productsupplierrefs_supplierref_id) references supplierref(supplierref_id));
		CREATE UNIQUE INDEX IF NOT EXISTS idx_productsupplierrefs ON productsupplierrefs(productsupplierrefs_product_id, productsupplierrefs_supplierref_id);
		
		CREATE TABLE IF NOT EXISTS producttags (
			producttags_product_id integer NOT NULL,
			producttags_tag_id integer NOT NULL,
			PRIMARY KEY(producttags_product_id, producttags_tag_id),
			FOREIGN KEY(producttags_product_id) references product(product_id),
			FOREIGN KEY(producttags_tag_id) references tag(tag_id));
		CREATE UNIQUE INDEX IF NOT EXISTS idx_producttags ON producttags(producttags_product_id, producttags_tag_id);

		CREATE TABLE IF NOT EXISTS new_storage (
			storage_id integer PRIMARY KEY,
			storage_creationdate datetime NOT NULL,
			storage_modificationdate datetime NOT NULL,
			storage_entrydate datetime,
			storage_exitdate datetime,
			storage_openingdate datetime,
			storage_expirationdate datetime,
			storage_quantity float,
			storage_barecode string,
			storage_comment string,
			storage_reference string,
			storage_batchnumber string,
			storage_todestroy boolean default 0,
			storage_archive boolean default 0,
			storage_qrcode blob,
			storage_concentration integer,
			person integer NOT NULL,
			product integer NOT NULL,
			storelocation integer NOT NULL,
			unit_concentration integer,
			unit_quantity integer,
			supplier integer,
			storage integer,
			FOREIGN KEY(unit_concentration) references unit(unit_id),
			FOREIGN KEY(storage) references storage(storage_id),
			FOREIGN KEY(unit_quantity) references unit(unit_id),
			FOREIGN KEY(supplier) references supplier(supplier_id),
			FOREIGN KEY(person) references person(person_id),
			FOREIGN KEY(product) references product(product_id),
			FOREIGN KEY(storelocation) references storelocation(storelocation_id));

		CREATE TABLE IF NOT EXISTS new_product (
			product_id integer PRIMARY KEY,
			product_specificity string,
			product_msds string,
			product_restricted boolean default 0,
			product_radioactive boolean default 0,
			product_threedformula string,
			product_molformula blob,
			product_disposalcomment string,
			product_remark string,
			product_qrcode string,
			product_sheet string,
			product_concentration integer,
			product_temperature integer,
			casnumber integer,
			cenumber integer,
			person integer NOT NULL,
			empiricalformula integer,
			linearformula integer,
			physicalstate integer,
			signalword integer,
			name integer NOT NULL,
			producerref integer,
			unit_temperature integer,
			category integer,
			FOREIGN KEY(unit_temperature) references unit(unit_id),
			FOREIGN KEY(producerref) references producerref(producerref_id),
			FOREIGN KEY(category) references category(category_id),
			FOREIGN KEY(casnumber) references casnumber(casnumber_id),
			FOREIGN KEY(cenumber) references cenumber(cenumber_id),
			FOREIGN KEY(person) references person(person_id),
			FOREIGN KEY(empiricalformula) references empiricalformula(empiricalformula_id),
			FOREIGN KEY(linearformula) references linearformula(linearformula_id),
			FOREIGN KEY(physicalstate) references physicalstate(physicalstate_id),
			FOREIGN KEY(signalword) references signalword(signalword_id),
			FOREIGN KEY(name) references name(name_id));

		INSERT INTO new_product (
			product_id,
			product_specificity,
			product_msds,
			product_restricted,
			product_radioactive,
			product_threedformula,
			product_molformula,
			product_disposalcomment,
			product_remark,
			product_qrcode,
			casnumber,
			cenumber,
			person,
			empiricalformula,
			linearformula,
			physicalstate,
			signalword,
			name
		)
		SELECT product_id,
			product_specificity,
			product_msds,
			product_restricted,
			product_radioactive,
			product_threedformula,
			product_molformula,
			product_disposalcomment,
			product_remark,
			product_qrcode,
			casnumber,
			cenumber,
			person,
			empiricalformula,
			linearformula,
			physicalstate,
			signalword,
			name
		FROM product;
		
		INSERT INTO new_storage (
			storage_id,
			storage_creationdate,
			storage_modificationdate,
			storage_entrydate,
			storage_exitdate,
			storage_openingdate,
			storage_expirationdate,
			storage_quantity,
			storage_barecode,
			storage_comment,
			storage_reference,
			storage_batchnumber,
			storage_todestroy,
			storage_archive,
			storage_qrcode,
			person,
			product,
			storelocation,
			unit_quantity,
			supplier,
			storage
		)
		SELECT storage_id,
			storage_creationdate,
			storage_modificationdate,
			storage_entrydate,
			storage_exitdate,
			storage_openingdate,
			storage_expirationdate,
			storage_quantity,
			storage_barecode,
			storage_comment,
			storage_reference,
			storage_batchnumber,
			storage_todestroy,
			storage_archive,
			storage_qrcode,
			person,
			product,
			storelocation,
			unit,
			supplier,
			storage
		FROM storage;

		DROP TABLE product;
		ALTER TABLE new_product RENAME TO product; 

		CREATE UNIQUE INDEX IF NOT EXISTS idx_product_casnumber ON product(product_id, casnumber);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_product_cenumber ON product(product_id, cenumber);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_product_empiricalformula ON product(product_id, empiricalformula);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_product_name ON product(product_id, name);

		DROP TABLE storage;
		ALTER TABLE new_storage RENAME TO storage; 

		CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_product ON storage(storage_id, product);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_storelocation ON storage(storage_id, storelocation);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_storelocation_product ON storage(storage_id, storelocation, product);
	

		UPDATE product SET empiricalformula=null WHERE empiricalformula=(SELECT empiricalformula_id FROM empiricalformula WHERE empiricalformula_label="XXXX");
		DELETE FROM empiricalformula where empiricalformula_label="XXXX";

		UPDATE product SET casnumber=null WHERE casnumber=(SELECT casnumber_id FROM casnumber WHERE casnumber_label="0000-00-0");
		DELETE FROM casnumber where casnumber_label="0000-00-0";

		CREATE INDEX "idx_permission_person" ON "permission" (
			"person" ASC
		);
		CREATE INDEX "idx_permission_perm_name" ON "permission" (
			"permission_perm_name"	ASC
		);
		CREATE INDEX "idx_permission_item_name" ON "permission" (
			"permission_item_name"	ASC
		);
		CREATE INDEX "idx_permission_entity_id" ON "permission" (
			"permission_entity_id"	ASC
		);

		PRAGMA user_version=3;
		COMMIT;
		PRAGMA foreign_keys=on;
		`
		globals.Log.Info("  migrating to user_version 3")
		if _, err = db.Exec(migrationThree); err != nil {
			return err
		}
	}

	// welcome announce
	if err = db.Get(&c, `SELECT count(*) FROM welcomeannounce`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting welcome announce")
		if _, err = db.Exec(inswelcomeannounce); err != nil {
			return err
		}
	}

	// symbols
	if err = db.Get(&c, `SELECT count(*) FROM symbol`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting symbols")
		if _, err = db.Exec(inssymbol); err != nil {
			return err
		}
	}

	// signal words
	if err = db.Get(&c, `SELECT count(*) FROM signalword`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting signal words")
		if _, err = db.Exec(inssignalword); err != nil {
			return err
		}
	}

	// signal units
	// now inserted in migration version 3
	// if err = db.Get(&c, `SELECT count(*) FROM unit`); err != nil {
	// 	return err
	// }
	// if c == 0 {
	// 	globals.Log.Info("  inserting units")
	// 	if _, err = db.Exec(insunit); err != nil {
	// 		return err
	// 	}
	// }

	// zero cas number
	// if err = db.Get(&c, `SELECT count(*) FROM casnumber`); err != nil {
	// 	return err
	// }
	// if c == 0 {
	// 	globals.Log.Info("  inserting zero cas number")
	// 	if _, err = db.Exec(`INSERT INTO casnumber (casnumber_label) VALUES ("0000-00-0")`); err != nil {
	// 		return err
	// 	}
	// }

	// cas numbers
	if err = db.Get(&c, `SELECT count(*) FROM casnumber`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting CMRs")
		r = csv.NewReader(strings.NewReader(data.CMR_CAS))
		r.Comma = ','
		if records, err = r.ReadAll(); err != nil {
			return err
		}
		for _, record := range records {
			if _, err = db.Exec(`INSERT INTO casnumber (casnumber_label, casnumber_cmr) VALUES (?, ?)`, record[0], record[1]); err != nil {
				return err
			}
		}
	}

	// tags
	if err = db.Get(&c, `SELECT count(*) FROM tag`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting tags")
		r = csv.NewReader(strings.NewReader(data.TAG))
		r.Comma = ','
		if records, err = r.ReadAll(); err != nil {
			return err
		}
		for _, record := range records {
			if _, err = db.Exec(`INSERT INTO tag (tag_label) VALUES (?)`, record[0]); err != nil {
				return err
			}
		}
	}

	// categories
	if err = db.Get(&c, `SELECT count(*) FROM category`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting categories")
		r = csv.NewReader(strings.NewReader(data.CATEGORY))
		r.Comma = ';'
		if records, err = r.ReadAll(); err != nil {
			return err
		}
		for _, record := range records {
			if _, err = db.Exec(`INSERT INTO category (category_label) VALUES (?)`, record[0]); err != nil {
				return err
			}
		}
	}

	// suppliers
	if err = db.Get(&c, `SELECT count(*) FROM supplier`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting suppliers")
		r = csv.NewReader(strings.NewReader(data.SUPPLIER))
		r.Comma = ','
		if records, err = r.ReadAll(); err != nil {
			return err
		}
		for _, record := range records {
			if _, err = db.Exec(`INSERT INTO supplier (supplier_label) VALUES (?)`, record[0]); err != nil {
				return err
			}
		}
	}

	// producers
	if err = db.Get(&c, `SELECT count(*) FROM producer`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting producers")
		r = csv.NewReader(strings.NewReader(data.PRODUCER))
		r.Comma = ','
		if records, err = r.ReadAll(); err != nil {
			return err
		}
		for _, record := range records {
			if _, err = db.Exec(`INSERT INTO producer (producer_label) VALUES (?)`, record[0]); err != nil {
				return err
			}
		}
	}

	// hazard statements
	if err = db.Get(&c, `SELECT count(*) FROM hazardstatement`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting hazard statements")
		r = csv.NewReader(strings.NewReader(data.HAZARDSTATEMENT))
		r.Comma = '\t'
		if records, err = r.ReadAll(); err != nil {
			return err
		}
		for _, record := range records {
			if _, err = db.Exec(`INSERT INTO hazardstatement (hazardstatement_label, hazardstatement_reference, hazardstatement_cmr) VALUES (?, ?, ?)`, record[0], record[1], record[2]); err != nil {
				return err
			}
		}
	}

	// precautionary statements
	if err = db.Get(&c, `SELECT count(*) FROM precautionarystatement`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting precautionary statements")
		r = csv.NewReader(strings.NewReader(data.PRECAUTIONARYSTATEMENT))
		r.Comma = '\t'
		if records, err = r.ReadAll(); err != nil {
			return err
		}
		for _, record := range records {
			if _, err = db.Exec(`INSERT INTO precautionarystatement (precautionarystatement_label, precautionarystatement_reference) VALUES (?, ?)`, record[0], record[1]); err != nil {
				return err
			}
		}
	}

	// zero empirical formula
	// if err = db.Get(&c, `SELECT count(*) FROM empiricalformula`); err != nil {
	// 	return err
	// }
	// if c == 0 {
	// 	globals.Log.Info("  inserting zero empirical formula")
	// 	if _, err = db.Exec(`INSERT INTO empiricalformula (empiricalformula_label) VALUES ("XXXX")`); err != nil {
	// 		return err
	// 	}
	// }

	// inserting default admin
	var admin *Person
	if err = db.Get(&c, `SELECT count(*) FROM person`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting admin user")
		admin = &Person{PersonEmail: "admin@chimitheque.fr", Permissions: []*Permission{&Permission{PermissionPermName: "all", PermissionItemName: "all", PermissionEntityID: -1}}}
		admin.PersonID, _ = db.CreatePerson(*admin)
		admin.PersonPassword = "chimitheque"
		if err = db.UpdatePersonPassword(*admin); err != nil {
			return err
		}
	}

	// inserting sample entity
	if err = db.Get(&c, `SELECT count(*) FROM entity`); err != nil {
		return err
	}
	if c == 0 {
		globals.Log.Info("  inserting sample entity")
		sentity := Entity{EntityName: "sample entity", EntityDescription: "you can delete me, I am just a sample entity", Managers: []*Person{admin}}
		if _, err = db.CreateEntity(sentity); err != nil {
			return err
		}
	}

	// tables creation
	globals.Log.Info("  vacuuming database")
	if _, err = db.Exec("VACUUM;"); err != nil {
		return err
	}

	return nil
}

func (db *SQLiteDataStore) Maintenance() {

	var (
		err  error
		sqlr string
		tx   *sql.Tx
	)

	//
	// Cleaning up casnumber labels duplicates.
	//
	if tx, err = db.Begin(); err != nil {
		globals.Log.Error(err)
		return
	}

	var casNumbers []CasNumber
	sqlr = `SELECT casnumber_id, casnumber_label FROM casnumber;`
	if err = db.Select(&casNumbers, sqlr); err != nil {
		globals.Log.Error(err)
		return
	}

	for _, casNumber := range casNumbers {

		if strings.HasPrefix(casNumber.CasNumberLabel.String, " ") || strings.HasSuffix(casNumber.CasNumberLabel.String, " ") {
			globals.Log.Infof("casnumber %s contains spaces", casNumber.CasNumberLabel)

			trimmedLabel := strings.Trim(casNumber.CasNumberLabel.String, " ")

			// Checking if the trimmed label already exists.
			var existCasNumber CasNumber
			sqlr = `SELECT casnumber_id, casnumber_label FROM casnumber WHERE casnumber_label=?;`
			if err = db.Get(&existCasNumber, sqlr, trimmedLabel); err != nil {
				switch err {
				case sql.ErrNoRows:
					// Just fixing the label.
					globals.Log.Info("  - fixing it")
					sqlr = `UPDATE casnumber SET casnumber_label=? WHERE casnumber_id=?;`
					if _, err = tx.Exec(sqlr, trimmedLabel, casNumber.CasNumberID); err != nil {
						globals.Log.Error(err)
						if errr := tx.Rollback(); errr != nil {
							globals.Log.Error(err)
							return
						}
						return
					}
					continue
				default:
					globals.Log.Error(err)
					return
				}
			}

			// Updating products with the found casnumber.
			globals.Log.Infof("  - correct cas number found, replacing it: %d -> %d", existCasNumber.CasNumberID, casNumber.CasNumberID)
			sqlr = `UPDATE product SET casnumber=? WHERE casnumber=?;`
			if _, err = tx.Exec(sqlr, existCasNumber.CasNumberID, casNumber.CasNumberID); err != nil {
				globals.Log.Error(err)
				if errr := tx.Rollback(); errr != nil {
					globals.Log.Error(err)
					return
				}
				return
			}

			// Deleting the wrong cas number.
			globals.Log.Info("  - deleting it")
			sqlr = `DELETE FROM casnumber WHERE casnumber_id=?;`
			if _, err = tx.Exec(sqlr, casNumber.CasNumberID); err != nil {
				globals.Log.Error(err)
				if errr := tx.Rollback(); errr != nil {
					globals.Log.Error(err)
					return
				}
				return
			}

		}

	}

	if err = tx.Commit(); err != nil {
		globals.Log.Error(err)
		if errr := tx.Rollback(); errr != nil {
			globals.Log.Error(errr)
			return
		}
	}

}

// Import import data from another Chimithèque instance
func (db *SQLiteDataStore) Import(url string) error {

	type r struct {
		Rows  []Product `json:"rows"`
		Total int       `json:"total"`
	}

	var (
		err         error
		httpresp    *http.Response
		bodyresp    r
		admin       Person
		notimported int
	)

	globals.Log.Info("- gathering remote products from " + url + "/e/products")
	if httpresp, err = http.Get(url + "/e/products"); err != nil {
		globals.Log.Error("can not get remote products " + err.Error())
	}
	defer httpresp.Body.Close()

	globals.Log.Info("- decoding response")
	if err = json.NewDecoder(httpresp.Body).Decode(&bodyresp); err != nil {
		globals.Log.Error("can not decode remote response " + err.Error())
	}
	globals.Log.Info(fmt.Sprintf("  found %d products", bodyresp.Total))

	globals.Log.Info("- retrieving default admin")
	if admin, err = db.GetPersonByEmail("admin@chimitheque.fr"); err != nil {
		globals.Log.Error("can not get default admin " + err.Error())
		os.Exit(1)
	}

	globals.Log.Info("- starting import")
	for _, p := range bodyresp.Rows {

		// cas number already exist ?
		if p.CasNumberID.Valid {
			var casnumber CasNumber
			if casnumber, err = db.GetProductsCasNumberByLabel(p.CasNumberLabel.String); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product cas number " + err.Error())
					os.Exit(1)
				}
			}
			// new cas number
			if casnumber == (CasNumber{}) {
				// setting cas number id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.CasNumber.CasNumberID = sql.NullInt64{Valid: true, Int64: -1}
			} else {
				// do not insert products with existing cas number
				notimported++
				continue
			}
		}

		// ce number already exist ?
		if p.CeNumberID.Valid {
			var cenumber CeNumber
			if cenumber, err = db.GetProductsCeNumberByLabel(p.CeNumberLabel.String); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product ce number " + err.Error())
					os.Exit(1)
				}
			}
			// new ce number
			if cenumber == (CeNumber{}) {
				// setting ce number id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.CeNumber.CeNumberID = sql.NullInt64{Valid: true, Int64: -1}
			} else {
				p.CeNumber = cenumber
			}
		}

		// empirical formula already exist ?
		if p.EmpiricalFormula.EmpiricalFormulaID.Valid {
			var eformula EmpiricalFormula
			if eformula, err = db.GetProductsEmpiricalFormulaByLabel(p.EmpiricalFormula.EmpiricalFormulaLabel.String); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product empirical formula " + err.Error())
					os.Exit(1)
				}
			}
			// new empirical formula
			if eformula == (EmpiricalFormula{}) {
				// setting empirical formula id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.EmpiricalFormula.EmpiricalFormulaID = sql.NullInt64{Valid: true, Int64: -1}
			} else {
				p.EmpiricalFormula = eformula
			}
		}

		// linear formula already exist ?
		if p.LinearFormula.LinearFormulaID.Valid {
			var lformula LinearFormula
			if lformula, err = db.GetProductsLinearFormulaByLabel(p.LinearFormula.LinearFormulaLabel.String); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product linear formula " + err.Error())
					os.Exit(1)
				}
			}
			// new linear formula
			if lformula == (LinearFormula{}) {
				// setting linear formula id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.LinearFormula.LinearFormulaID = sql.NullInt64{Valid: true, Int64: -1}
			} else {
				p.LinearFormula = lformula
			}
		}

		// physical state already exist ?
		if p.PhysicalState.PhysicalStateID.Valid {
			var physicalstate PhysicalState
			if physicalstate, err = db.GetProductsPhysicalStateByLabel(p.PhysicalState.PhysicalStateLabel.String); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product physical state " + err.Error())
					os.Exit(1)
				}
			}
			// new physical state
			if physicalstate == (PhysicalState{}) {
				// setting physical state id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.PhysicalState.PhysicalStateID = sql.NullInt64{Valid: true, Int64: -1}
			} else {
				p.PhysicalState = physicalstate
			}
		}

		// signal word already exist ?
		if p.SignalWord.SignalWordID.Valid {
			var signalword SignalWord
			if signalword, err = db.GetProductsSignalWordByLabel(p.SignalWord.SignalWordLabel.String); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product signal word " + err.Error())
					os.Exit(1)
				}
			}
			// new signal word
			if signalword == (SignalWord{}) {
				// setting signal word id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.SignalWord.SignalWordID = sql.NullInt64{Valid: true, Int64: -1}
			} else {
				p.SignalWord = signalword
			}
		}

		// name already exist ?
		var name Name
		if name, err = db.GetProductsNameByLabel(p.Name.NameLabel); err != nil {
			if err != sql.ErrNoRows {
				globals.Log.Error("can not get product name " + err.Error())
				os.Exit(1)
			}
		}
		// new name
		if name == (Name{}) {
			// setting name id to -1 for the CreateProduct method
			// to automatically insert it into the db
			p.Name.NameID = -1
		} else {
			p.Name = name
		}

		// synonyms
		var (
			processedSyn map[string]string
			newSyn       []Name
			ok           bool
		)
		// duplicate names map
		processedSyn = make(map[string]string)
		processedSyn[p.Name.NameLabel] = ""
		for _, syn := range p.Synonyms {
			// duplicates hunting
			if _, ok = processedSyn[syn.NameLabel]; ok {
				globals.Log.Debug("leaving duplicate synonym " + syn.NameLabel)
				continue
			}

			processedSyn[syn.NameLabel] = ""

			// synonym already exist ?
			var syn2 Name
			if syn2, err = db.GetProductsNameByLabel(syn.NameLabel); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product synonym " + err.Error())
					os.Exit(1)
				}
			}
			// new synonym
			if syn2 == (Name{}) {
				// setting synonym id to -1 for the CreateProduct method
				// to automatically insert it into the db
				newSyn = append(newSyn, Name{NameID: -1, NameLabel: syn.NameLabel})
			} else {
				newSyn = append(newSyn, syn2)
			}
		}
		p.Synonyms = newSyn

		// classes of compounds
		for i, coc := range p.ClassOfCompound {
			// class of compounds already exist ?
			var coc2 ClassOfCompound
			if coc2, err = db.GetProductsClassOfCompoundByLabel(coc.ClassOfCompoundLabel); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product class of compounds " + err.Error())
					os.Exit(1)
				}
			}
			// new class of compounds
			if coc2 == (ClassOfCompound{}) {
				// setting class of compounds id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.ClassOfCompound[i].ClassOfCompoundID = -1
			} else {
				p.ClassOfCompound[i] = coc2
			}
		}

		// symbols
		for i, sym := range p.Symbols {
			// symbols already exist ?
			var sym2 Symbol
			if sym2, err = db.GetProductsSymbolByLabel(sym.SymbolLabel); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product symbol " + err.Error())
					os.Exit(1)
				}
			}
			// new symbol
			if sym2 == (Symbol{}) {
				// setting symbol id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.Symbols[i].SymbolID = -1
			} else {
				p.Symbols[i] = sym2
			}
		}

		// hazard statements
		for i, hs := range p.HazardStatements {
			// hazard statement already exist ?
			var hs2 HazardStatement
			if hs2, err = db.GetProductsHazardStatementByReference(hs.HazardStatementReference); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product hazard statement " + err.Error())
					os.Exit(1)
				}
			}
			// new hazard statement
			if hs2 == (HazardStatement{}) {
				// setting hazard statement id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.HazardStatements[i].HazardStatementID = -1
			} else {
				p.HazardStatements[i] = hs2
			}
		}

		// precautionnary statements
		for i, ps := range p.PrecautionaryStatements {
			// precautionary statement already exist ?
			var ps2 PrecautionaryStatement
			if ps2, err = db.GetProductsPrecautionaryStatementByReference(ps.PrecautionaryStatementReference); err != nil {
				if err != sql.ErrNoRows {
					globals.Log.Error("can not get product precautionary statement " + err.Error())
					os.Exit(1)
				}
			}
			// new precautionary statement
			if ps2 == (PrecautionaryStatement{}) {
				// setting precautionary statement id to -1 for the CreateProduct method
				// to automatically insert it into the db
				p.PrecautionaryStatements[i].PrecautionaryStatementID = -1
			} else {
				p.PrecautionaryStatements[i] = ps2
			}
		}

		// setting default admin as creator
		p.Person = admin

		// finally creating the product
		if _, err = db.CreateProduct(p); err != nil {
			globals.Log.Error("can not create product " + err.Error())
			os.Exit(1)
		}

	}

	globals.Log.Info(fmt.Sprintf("%d products not imported (duplicates)", notimported))

	return nil
}

// ImportV1 import data from CSV
func (db *SQLiteDataStore) ImportV1(dir string) error {

	var (
		csvFile *os.File
		//csvReader *csv.Reader
		csvMap []map[string]string
		err    error
		res    sql.Result
		lastid int64
		c      int      // count result
		tx     *sqlx.Tx // db transaction
		sqlr   string   // sql request

		zerocasnumberid        int
		zeroempiricalformulaid int
		zeropersonid           int // admin id
		zerohsid               string
		zeropsid               string

		// ids mappings
		// O:old N:new R:reverse
		mONperson        map[string]string   // oldid <> newid map for user table
		mONsupplier      map[string]string   // oldid <> newid map for supplier table
		mONunit          map[string]string   // oldid <> newid map for unit table
		mONentity        map[string]string   // oldid <> newid map for entity table
		mONstorelocation map[string]string   // oldid <> newid map for storelocation table
		mOOentitypeople  map[string][]string // managers, oldentityid <> oldpersonid
		mRNNcasnumber    map[string]string   // newlabel <> newid
		mRNNcenumber     map[string]string   // newlabel <> newid

		mONproduct                map[string]string // oldid <> newid map for product table
		mONclassofcompound        map[string]string // oldid <> newid map for classofcompound table
		mONempiricalformula       map[string]string // oldid <> newid map for empiricalformula table
		mONlinearformula          map[string]string // oldid <> newid map for linearformula table
		mONname                   map[string]string // oldid <> newid map for name table
		mONphysicalstate          map[string]string // oldid <> newid map for physicalstate table
		mONhazardstatement        map[string]string // oldid <> newid map for hazardstatement table
		mONprecautionarystatement map[string]string // oldid <> newid map for precautionarystatement table
		mONsymbol                 map[string]string // oldid <> newid map for symbol table
		mONsignalword             map[string]string // oldid <> newid map for signalword table

	)

	// init maps
	mONproduct = make(map[string]string)
	mONperson = make(map[string]string)
	mONunit = make(map[string]string)
	mONsupplier = make(map[string]string)
	mONentity = make(map[string]string)
	mONstorelocation = make(map[string]string)
	mOOentitypeople = make(map[string][]string)
	mRNNcasnumber = make(map[string]string)
	mRNNcenumber = make(map[string]string)
	mONclassofcompound = make(map[string]string)
	mONempiricalformula = make(map[string]string)
	mONlinearformula = make(map[string]string)
	mONname = make(map[string]string)
	mONphysicalstate = make(map[string]string)
	mONhazardstatement = make(map[string]string)
	mONprecautionarystatement = make(map[string]string)
	mONsymbol = make(map[string]string)
	mONsignalword = make(map[string]string)

	// number regex
	rnumber := regexp.MustCompile("([0-9]+)")

	// checking tables empty
	if err = db.Get(&c, `SELECT count(*) FROM product`); err != nil {
		return err
	}
	if c != 0 {
		panic("person product not empty - can not import")
	}

	// beginning transaction
	if tx, err = db.Beginx(); err != nil {
		return err
	}

	//
	// entity
	//
	globals.Log.Info("- importing entity")
	rentityName := regexp.MustCompile("user_[0-9]+|root_entity|all_entity")
	if csvFile, err = os.Open(path.Join(dir, "entity.csv")); err != nil {
		return (err)
	}

	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		role := k["role"]
		description := k["description"]
		manager := k["manager"]

		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	role := line[1]
		// 	description := line[2]
		// 	manager := line[3]

		// finding web2py like manager ids
		ms := rnumber.FindAllString(manager, -1)
		for _, m := range ms {
			// leaving hardcoded zeros
			if m != "0" {
				mOOentitypeople[id] = append(mOOentitypeople[id], m)
				globals.Log.Debug("entity with old id " + id + " has manager with old id " + m)
			}
		}

		// leaving web2py specific entries
		if !rentityName.MatchString(role) {
			globals.Log.Debug("  " + role)
			sqlr = `INSERT INTO entity(entity_name, entity_description) VALUES (?, ?)`
			if res, err = tx.Exec(sqlr, role, description); err != nil {
				globals.Log.Error("error importing entity " + role)
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			// getting the last inserted id
			if lastid, err = res.LastInsertId(); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			// populating the map
			mONentity[id] = strconv.FormatInt(lastid, 10)
			globals.Log.Debug("entity with old id " + id + " has new  id " + strconv.FormatInt(lastid, 10))
		}
	}

	//
	// storelocation
	//
	globals.Log.Info("- importing store locations")
	if csvFile, err = os.Open(path.Join(dir, "store_location.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		entity := k["entity"]
		parent := k["parent"]
		canStore := k["can_store"]
		color := k["color"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]
		// 	entity := line[2]
		// 	parent := line[3]
		// 	canStore := false
		// 	if line[4] == "T" {
		// 		canStore = true
		// 	}
		// 	color := line[5]

		newentity := mONentity[entity]
		newparent := sql.NullString{}
		np := mONstorelocation[parent]
		if np != "" {
			newparent = sql.NullString{Valid: true, String: np}
		}
		newcanStore := false
		if canStore == "T" {
			newcanStore = true
		}
		globals.Log.Debug("storelocation " + label + ", entity:" + newentity + ", parent:" + newparent.String)
		sqlr = `INSERT INTO storelocation(storelocation_name, storelocation_color, storelocation_canstore, storelocation_fullpath, entity, storelocation) VALUES (?, ?, ?, ?, ?, ?)`
		if res, err = tx.Exec(sqlr, label, color, newcanStore, "", newentity, newparent); err != nil {
			globals.Log.Error("error importing storelocation " + label)
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// getting the last inserted id
		if lastid, err = res.LastInsertId(); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// populating the map
		mONstorelocation[id] = strconv.FormatInt(lastid, 10)
	}

	//
	// person
	//
	globals.Log.Info("- importing user")
	if csvFile, err = os.Open(path.Join(dir, "person.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		email := k["email"]
		password := k["password"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	email := line[3]
		// 	password := utils.RandStringBytes(64)

		sqlr = `INSERT INTO person(person_email, person_password) VALUES (?, ?)`
		if res, err = tx.Exec(sqlr, email, password); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// getting the last inserted id
		if lastid, err = res.LastInsertId(); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// populating the map
		mONperson[id] = strconv.FormatInt(lastid, 10)
	}

	//
	// permissions
	//
	globals.Log.Info("- initializing default permissions (r products)")
	for _, newpid := range mONperson {
		sqlr = `INSERT INTO permission(person, permission_perm_name, permission_item_name, permission_entity_id) VALUES (?, ?, ?, ?)`
		if _, err = tx.Exec(sqlr, newpid, "r", "products", -1); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
	}

	//
	// managers
	//
	globals.Log.Info("- importing managers")
	for oldentityid, oldmanagerids := range mOOentitypeople {
		for _, oldmanagerid := range oldmanagerids {
			newentityid := mONentity[oldentityid]
			newmanagerid := mONperson[oldmanagerid]
			// silently missing entities with no managers
			if newmanagerid != "" {
				sqlr = `INSERT INTO entitypeople(entitypeople_entity_id, entitypeople_person_id) VALUES (?, ?)`
				if _, err = tx.Exec(sqlr, newentityid, newmanagerid); err != nil {
					if errr := tx.Rollback(); errr != nil {
						return errr
					}
					return err
				}
				globals.Log.Debug("person "+newmanagerid+", permission_perm_name: all permission_item_name: all", " permission_entity_id:"+newentityid)
				sqlr = `INSERT INTO permission(person, permission_perm_name, permission_item_name, permission_entity_id) VALUES (?, ?, ?, ?)`
				if _, err = tx.Exec(sqlr, newmanagerid, "all", "all", newentityid); err != nil {
					if errr := tx.Rollback(); errr != nil {
						return errr
					}
					return err
				}
			}
		}
	}

	//
	// membership
	//
	globals.Log.Info("- importing membership")
	if csvFile, err = os.Open(path.Join(dir, "membership.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		userId := k["user_id"]
		groupId := k["group_id"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	userId := line[1]
		// 	groupId := line[2]
		newuserId := mONperson[userId]
		newgroupId := mONentity[groupId]

		if newuserId != "" && newgroupId != "" {
			sqlr = `INSERT INTO personentities(personentities_person_id, personentities_entity_id) VALUES (?, ?)`
			if _, err = tx.Exec(sqlr, newuserId, newgroupId); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			sqlr = `INSERT INTO permission(person, permission_perm_name, permission_item_name, permission_entity_id) VALUES (?, ?, ?, ?)`
			if _, err = tx.Exec(sqlr, newuserId, "r", "entities", newgroupId); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
		}
	}

	//
	// class of compounds
	//
	globals.Log.Info("- importing classes of compounds")
	if csvFile, err = os.Open(path.Join(dir, "class_of_compounds.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]

		sqlr = `INSERT INTO classofcompound(classofcompound_id, classofcompound_label) VALUES (?, ?)`
		if res, err = tx.Exec(sqlr, id, label); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// getting the last inserted id
		if lastid, err = res.LastInsertId(); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// populating the map
		mONclassofcompound[id] = strconv.FormatInt(lastid, 10)
	}

	//
	// empirical formula
	//
	globals.Log.Info("- importing empirical formulas")
	if csvFile, err = os.Open(path.Join(dir, "empirical_formula.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]
		if label == "----" {
			continue
		}

		sqlr = `INSERT INTO empiricalformula(empiricalformula_label) VALUES (?)`
		if res, err = tx.Exec(sqlr, label); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// getting the last inserted id
		if lastid, err = res.LastInsertId(); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// populating the map
		mONempiricalformula[id] = strconv.FormatInt(lastid, 10)
	}

	//
	// linear formula
	//
	globals.Log.Info("- importing linear formulas")
	if csvFile, err = os.Open(path.Join(dir, "linear_formula.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]
		if label == "----" {
			continue
		}

		sqlr = `INSERT INTO linearformula(linearformula_label) VALUES (?)`
		if res, err = tx.Exec(sqlr, label); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// getting the last inserted id
		if lastid, err = res.LastInsertId(); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// populating the map
		mONlinearformula[id] = strconv.FormatInt(lastid, 10)
	}

	//
	// name
	//
	globals.Log.Info("- importing product names")
	if csvFile, err = os.Open(path.Join(dir, "name.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]
		label = strings.Replace(label, "@", "_", -1)

		globals.Log.Debug("label:" + label)
		sqlr = `INSERT INTO name(name_id, name_label) VALUES (?, ?)`
		if res, err = tx.Exec(sqlr, id, label); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// getting the last inserted id
		if lastid, err = res.LastInsertId(); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// populating the maps
		mONname[id] = strconv.FormatInt(lastid, 10)
	}

	//
	// physical states
	//
	globals.Log.Info("- importing product physical states")
	if csvFile, err = os.Open(path.Join(dir, "physical_state.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]

		sqlr = `INSERT INTO physicalstate(physicalstate_id, physicalstate_label) VALUES (?, ?)`
		if res, err = tx.Exec(sqlr, id, label); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// getting the last inserted id
		if lastid, err = res.LastInsertId(); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// populating the map
		mONphysicalstate[id] = strconv.FormatInt(lastid, 10)
	}

	//
	// cas numbers
	//
	globals.Log.Info("- extracting and importing cas numbers from products")
	globals.Log.Info("  gathering existing CMR cas numbers")
	var (
		rows     *sql.Rows
		casid    string
		caslabel string
	)
	if rows, err = tx.Query(`SELECT casnumber_id, casnumber_label FROM casnumber`); err != nil {
		globals.Log.Error("error gathering existing CMR cas numbers")
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}
	for rows.Next() {
		err := rows.Scan(&casid, &caslabel)
		if err != nil {
			globals.Log.Fatal(err)
		}
		mRNNcasnumber[caslabel] = casid
	}
	if csvFile, err = os.Open(path.Join(dir, "product.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {

		casnumber := k["cas_number"]
		globals.Log.Debug(fmt.Sprintf("casnumber: %s", casnumber))
		if _, ok := mRNNcasnumber[casnumber]; !ok {
			sqlr = `INSERT INTO casnumber(casnumber_label) VALUES (?)`
			if res, err = tx.Exec(sqlr, casnumber); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			// getting the last inserted id
			if lastid, err = res.LastInsertId(); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			// populating the map
			mRNNcasnumber[casnumber] = strconv.FormatInt(lastid, 10)
		}
	}

	//
	// ce numbers
	//
	globals.Log.Info("- extracting and importing ce numbers from products")
	if csvFile, err = os.Open(path.Join(dir, "product.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {

		cenumber := k["ce_number"]
		if cenumber != "" {
			if _, ok := mRNNcenumber[cenumber]; !ok {
				sqlr = `INSERT INTO cenumber(cenumber_label) VALUES (?)`
				if res, err = tx.Exec(sqlr, cenumber); err != nil {
					if errr := tx.Rollback(); errr != nil {
						return errr
					}
					return err
				}
				// getting the last inserted id
				if lastid, err = res.LastInsertId(); err != nil {
					if errr := tx.Rollback(); errr != nil {
						return errr
					}
					return err
				}
				// populating the map
				mRNNcenumber[cenumber] = strconv.FormatInt(lastid, 10)
			}
		}
	}

	//
	// supplier
	//
	globals.Log.Info("- importing storage suppliers")
	if csvFile, err = os.Open(path.Join(dir, "supplier.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		if errr := tx.Rollback(); errr != nil {
		// 			return errr
		// 		}
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]

		globals.Log.Debug("label:" + label)
		sqlr = `INSERT INTO supplier(supplier_id, supplier_label) VALUES (?, ?)`
		if res, err = tx.Exec(sqlr, id, label); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// getting the last inserted id
		if lastid, err = res.LastInsertId(); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
		// populating the maps
		mONsupplier[id] = strconv.FormatInt(lastid, 10)
	}

	// committing changes
	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	// beginning new transaction
	if tx, err = db.Beginx(); err != nil {
		return err
	}

	//
	// products
	//
	globals.Log.Info("- importing products")
	globals.Log.Info("  retrieving zero empirical id")
	if err = db.Get(&zeroempiricalformulaid, `SELECT empiricalformula_id FROM empiricalformula WHERE empiricalformula_label = "XXXX"`); err != nil {
		globals.Log.Error("error retrieving zero empirical id")
		return err
	}
	globals.Log.Info("  retrieving zero casnumber id")
	if err = db.Get(&zerocasnumberid, `SELECT casnumber_id FROM casnumber WHERE casnumber_label = "0000-00-0"`); err != nil {
		globals.Log.Error("error retrieving zero casnumber id")
		return err
	}
	globals.Log.Info("  retrieving default admin id")
	if err = db.Get(&zeropersonid, `SELECT person_id FROM person WHERE person_email = "admin@chimitheque.fr"`); err != nil {
		globals.Log.Error("error retrieving default admin id")
		return err
	}
	globals.Log.Info("  gathering hazardstatement ids")
	if csvFile, err = os.Open(path.Join(dir, "hazard_statement.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		reference := k["reference"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		return err
		// 	}
		// 	id := line[0]
		// 	reference := line[2]
		if reference == "----" {
			zerohsid = id
			continue
		}
		// finding new id
		var nid int
		if err = db.Get(&nid, `SELECT hazardstatement_id FROM hazardstatement WHERE hazardstatement_reference = ?`, reference); err != nil {
			globals.Log.Info("no hazardstatement id for " + reference + " inserting a new one")
			var (
				res   sql.Result
				nid64 int64
			)
			if res, err = tx.Exec(`INSERT INTO hazardstatement (hazardstatement_label, hazardstatement_reference) VALUES (?, ?)`, id, reference); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			// getting the last inserted id
			if nid64, err = res.LastInsertId(); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			nid = int(nid64)
		}
		mONhazardstatement[id] = strconv.Itoa(nid)
	}
	globals.Log.Info("  gathering precautionarystatement ids")
	if csvFile, err = os.Open(path.Join(dir, "precautionary_statement.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		reference := k["reference"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		return err
		// 	}
		// 	id := line[0]
		// 	reference := line[2]
		if reference == "----" {
			zeropsid = id
			continue
		}
		// finding new id
		var nid int
		if err = db.Get(&nid, `SELECT precautionarystatement_id FROM precautionarystatement WHERE precautionarystatement_reference = ?`, reference); err != nil {
			globals.Log.Info("no precautionarystatement id for " + reference + " inserting a new one")
			var (
				res   sql.Result
				nid64 int64
			)
			if res, err = tx.Exec(`INSERT INTO precautionarystatement (precautionarystatement_label, precautionarystatement_reference) VALUES (?, ?)`, id, reference); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			// getting the last inserted id
			if nid64, err = res.LastInsertId(); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			nid = int(nid64)
		}
		mONprecautionarystatement[id] = strconv.Itoa(nid)
	}
	globals.Log.Info("  gathering symbol ids")
	if csvFile, err = os.Open(path.Join(dir, "symbol.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]
		// finding new id
		var nid int
		if err = db.Get(&nid, `SELECT symbol_id FROM symbol WHERE symbol_label = ?`, label); err != nil {
			globals.Log.Error("error gathering symbol id for " + label)
			return err
		}
		mONsymbol[id] = strconv.Itoa(nid)
	}
	globals.Log.Info("  gathering signalword ids")
	if csvFile, err = os.Open(path.Join(dir, "signal_word.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]
		// finding new id
		var nid int
		if err = db.Get(&nid, `SELECT signalword_id FROM signalword WHERE signalword_label = ?`, label); err != nil {
			globals.Log.Error("error gathering signalword id for " + label)
			return err
		}
		mONsignalword[id] = strconv.Itoa(nid)
	}

	if csvFile, err = os.Open(path.Join(dir, "product.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		cenumber := k["ce_number"]
		person := k["person"]
		name := k["name"]
		synonym := k["synonym"]
		restricted := k["restricted_access"]
		specificity := k["specificity"]
		tdformula := k["tdformula"]
		empiricalformula := k["empirical_formula"]
		linearformula := k["linear_formula"]
		msds := k["msds"]
		physicalstate := k["physical_state"]
		coc := k["class_of_compounds"]
		symbol := k["symbol"]
		signalword := k["signal_word"]
		hazardstatement := k["hazard_statement"]
		precautionarystatement := k["precautionary_statement"]
		disposalcomment := k["disposal_comment"]
		remark := k["remark"]
		archive := k["archive"]
		casnumber := k["cas_number"]
		isradio := k["is_radio"]

		newperson := mONperson[person]
		if newperson == "" {
			newperson = strconv.Itoa(zeropersonid)
		}
		newname := mONname[name]
		newrestricted := false
		if restricted == "T" {
			newrestricted = true
		}
		newspecificity := specificity
		newtdformula := tdformula
		newempiricalformula := mONempiricalformula[empiricalformula]
		if newempiricalformula == "" {
			newempiricalformula = strconv.Itoa(zeroempiricalformulaid)
		}
		newlinearformula := sql.NullInt64{}
		if mONlinearformula[linearformula] != "" {
			i, e := strconv.ParseInt(mONlinearformula[linearformula], 10, 64)
			if e != nil {
				globals.Log.Error("error converting linearformula id for " + mONlinearformula[linearformula])
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			newlinearformula = sql.NullInt64{Valid: true, Int64: i}
		}
		newmsds := msds
		newphysicalstate := sql.NullInt64{}
		if mONphysicalstate[physicalstate] != "" {
			i, e := strconv.ParseInt(mONphysicalstate[physicalstate], 10, 64)
			if e != nil {
				globals.Log.Error("error converting physicalstate id for " + mONphysicalstate[physicalstate])
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			newphysicalstate = sql.NullInt64{Valid: true, Int64: i}
		}
		newsignalword := sql.NullInt64{}
		if mONsignalword[signalword] != "" {
			i, e := strconv.ParseInt(mONsignalword[signalword], 10, 64)
			if e != nil {
				globals.Log.Error("error converting signalword id for " + mONsignalword[signalword])
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			newsignalword = sql.NullInt64{Valid: true, Int64: i}
		}
		newdisposalcomment := disposalcomment
		newremark := remark
		newarchive := false
		if archive == "T" {
			newarchive = true
		}
		newcasnumber := mRNNcasnumber[casnumber]
		if newcasnumber == "" {
			newcasnumber = strconv.Itoa(zerocasnumberid)
		}
		newcenumber := mRNNcenumber[cenumber]
		newisradio := false
		if isradio == "T" {
			newisradio = true
		}

		// do not import archived cards
		if !newarchive {
			reqValues := "?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?"
			reqArgs := []interface{}{
				newspecificity,
				newmsds,
				newrestricted,
				newisradio,
				newtdformula,
				newdisposalcomment,
				newremark,
				newempiricalformula,
				newlinearformula,
				newphysicalstate,
				newsignalword,
				newperson,
				newcasnumber,
				newname,
			}
			sqlr = `INSERT INTO product (product_specificity, 
                product_msds, 
                product_restricted, 
                product_radioactive, 
                product_threedformula, 
                product_disposalcomment, 
                product_remark,
                empiricalformula,
                linearformula,
                physicalstate,
                signalword,
                person,
				casnumber,
				name`
			if newcenumber != "" {
				sqlr += ",cenumber"
				reqValues += ",?"
				reqArgs = append(reqArgs, newcenumber)
			}
			sqlr += `) VALUES (` + reqValues + `)`

			globals.Log.Debug(fmt.Sprintf(`newperson: %s,
			newname: %s,
			newrestricted: %t,
			newspecificity: %s,
			newtdformula: %s,
			newempiricalformula: %s,
			newlinearformula: %v,
			newmsds: %s,
			newphysicalstate: %v,
			newsignalword: %v,
			newdisposalcomment: %s,
			newremark: %s,
			newarchive: %t,
			casnumber: %s,
			newcasnumber: %s,
			newcenumber: %s,
			newisradio: %t
			`, newperson,
				newname,
				newrestricted,
				newspecificity,
				newtdformula,
				newempiricalformula,
				newlinearformula,
				newmsds,
				newphysicalstate,
				newsignalword,
				newdisposalcomment,
				newremark,
				newarchive,
				casnumber,
				newcasnumber,
				newcenumber,
				newisradio))

			if res, err = tx.Exec(sqlr, reqArgs...); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			// getting the last inserted id
			if lastid, err = res.LastInsertId(); err != nil {
				globals.Log.Error("error importing product")
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
			// populating the map
			mONproduct[id] = strconv.FormatInt(lastid, 10)

			// coc
			cocs := rnumber.FindAllString(coc, -1)
			for _, c := range cocs {
				sqlr = `INSERT INTO productclassofcompound (productclassofcompound_product_id, productclassofcompound_classofcompound_id) VALUES (?,?)`
				if _, err = tx.Exec(sqlr, lastid, mONclassofcompound[c]); err != nil {
					// not leaving on errors
					globals.Log.Debug("non fatal error importing product class of compounds with id " + c + ": " + err.Error())
				}
			}
			// synonym
			syns := rnumber.FindAllString(synonym, -1)
			for _, s := range syns {
				if s == "0" {
					continue
				}
				// leaving hardcoded zeros
				sqlr = `INSERT INTO productsynonyms (productsynonyms_product_id, productsynonyms_name_id) VALUES (?,?)`
				if _, err = tx.Exec(sqlr, lastid, mONname[s]); err != nil {
					// not leaving on errors
					globals.Log.Debug("non fatal error importing product synonym with id " + s + ": " + err.Error())
				}
			}
			// symbol
			symbols := rnumber.FindAllString(symbol, -1)
			for _, s := range symbols {
				sqlr = `INSERT INTO productsymbols (productsymbols_product_id, productsymbols_symbol_id) VALUES (?,?)`
				if _, err = tx.Exec(sqlr, lastid, mONsymbol[s]); err != nil {
					// not leaving on errors
					globals.Log.Error("error importing product symbol with id " + s + ": " + err.Error())
					if errr := tx.Rollback(); errr != nil {
						return errr
					}
					return err
				}
			}
			// hs
			hss := rnumber.FindAllString(hazardstatement, -1)
			for _, s := range hss {
				if s == zerohsid {
					continue
				}
				sqlr = `INSERT INTO producthazardstatements (producthazardstatements_product_id, producthazardstatements_hazardstatement_id) VALUES (?,?)`
				if _, err = tx.Exec(sqlr, lastid, mONhazardstatement[s]); err != nil {
					// not leaving on errors
					globals.Log.Error("error importing product hazardstatement with id " + s + ": " + err.Error())
					if errr := tx.Rollback(); errr != nil {
						return errr
					}
					return err
				}
			}
			// ps
			pss := rnumber.FindAllString(precautionarystatement, -1)
			for _, s := range pss {
				if s == zeropsid {
					continue
				}
				sqlr = `INSERT INTO productprecautionarystatements (productprecautionarystatements_product_id, productprecautionarystatements_precautionarystatement_id) VALUES (?,?)`
				if _, err = tx.Exec(sqlr, lastid, mONprecautionarystatement[s]); err != nil {
					// not leaving on errors
					globals.Log.Error("error importing product precautionarystatement with id " + s + ": " + err.Error())
					if errr := tx.Rollback(); errr != nil {
						return errr
					}
					return err
				}
			}
		}

	}

	// committing changes
	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	// beginning new transaction
	if tx, err = db.Beginx(); err != nil {
		return err
	}

	//
	// storages
	//
	globals.Log.Info("- importing storages")
	globals.Log.Info("  gathering unit ids")
	if csvFile, err = os.Open(path.Join(dir, "unit.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		id := k["id"]
		label := k["label"]
		// csvReader = csv.NewReader(bufio.NewReader(csvFile))
		// i = 0
		// for {
		// 	line, error := csvReader.Read()

		// 	// skip header
		// 	if i == 0 {
		// 		i++
		// 		continue
		// 	}

		// 	if error == io.EOF {
		// 		break
		// 	} else if error != nil {
		// 		return err
		// 	}
		// 	id := line[0]
		// 	label := line[1]
		// uppercase liter
		label = strings.Replace(label, "l", "L", -1)
		// finding new id
		var nid int
		if err = db.Get(&nid, `SELECT unit_id FROM unit WHERE unit_label = ?`, label); err != nil {
			globals.Log.Error("error gathering unit id for " + label)
			return err
		}
		mONunit[id] = strconv.Itoa(nid)
	}

	if csvFile, err = os.Open(path.Join(dir, "storage.csv")); err != nil {
		return (err)
	}
	csvMap = utils.CSVToMap(bufio.NewReader(csvFile))
	for _, k := range csvMap {
		oldid := k["id"]
		product := k["product"]
		person := k["person"]
		storeLocation := k["store_location"]
		unit := k["unit"]
		entrydate := k["entry_datetime"]
		exitdate := k["exit_datetime"]
		comment := k["comment"]
		barecode := k["barecode"]
		reference := k["reference"]
		batchNumber := k["batch_number"]
		supplier := k["supplier"]
		archive := k["archive"]
		creationdate := k["creation_datetime"]
		volumeWeight := k["volume_weight"]
		openingdate := k["opening_datetime"]
		toDestroy := k["to_destroy"]
		expirationdate := k["expiration_datetime"]

		globals.Log.Debug(globals.Log.WithFields(logrus.Fields{
			"oldid":         oldid,
			"product":       product,
			"person":        person,
			"storeLocation": storeLocation,
			"unit":          unit,
			"entrydate":     entrydate,
			"exitdate":      exitdate,
			"supplier":      supplier,
		}))

		newproduct := mONproduct[product]
		newperson := mONperson[person]
		if newperson == "" {
			newperson = strconv.Itoa(zeropersonid)
		}
		newstoreLocation := mONstorelocation[storeLocation]
		newunit := mONunit[unit]
		var newentrydate *time.Time
		if entrydate != "" {
			newentrydate = &time.Time{}
			*newentrydate, _ = time.Parse("2006-01-02 15:04:05", entrydate)
		}
		var newexitdate *time.Time
		if exitdate != "" {
			newexitdate = &time.Time{}
			*newexitdate, _ = time.Parse("2006-01-02 15:04:05", exitdate)
		}
		newcomment := comment
		newbarecode := barecode
		newreference := reference
		newbatchNumber := batchNumber
		newsupplier := mONsupplier[supplier]
		newarchive := false
		if archive == "T" {
			newarchive = true
		}
		newstorageCreationdate := time.Now()
		if creationdate != "" {
			newstorageCreationdate, _ = time.Parse("2006-01-02 15:04:05", creationdate)
		}
		newvolumeWeight := volumeWeight
		if newvolumeWeight == "" {
			newvolumeWeight = "1"
		}
		var newopeningdate *time.Time
		if openingdate != "" {
			newopeningdate = &time.Time{}
			*newopeningdate, _ = time.Parse("2006-01-02 15:04:05", openingdate)
		}
		newtoDestroy := false
		if toDestroy == "T" {
			newtoDestroy = true
		}
		var newexpirationdate *time.Time
		if expirationdate != "" {
			newexpirationdate = &time.Time{}
			*newexpirationdate, _ = time.Parse("2006-01-02 15:04:05", expirationdate)
		}

		// do not import archived cards
		if !newarchive {
			reqValues := "?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?"
			reqArgs := []interface{}{
				newstorageCreationdate,
				newstorageCreationdate,
				newcomment,
				newreference,
				newbatchNumber,
				newvolumeWeight,
				newbarecode,
				newtoDestroy,
				newperson,
				newproduct,
				newstoreLocation,
			}
			sqlr = `INSERT INTO storage (storage_creationdate, 
                storage_modificationdate, 
                storage_comment, 
                storage_reference, 
                storage_batchnumber, 
                storage_quantity, 
                storage_barecode,
                storage_todestroy,
                person,
                product,
				storelocation`
			if newunit != "" {
				sqlr += ",unit"
				reqValues += ",?"
				reqArgs = append(reqArgs, newunit)
			}
			if newsupplier != "" {
				sqlr += ",supplier"
				reqValues += ",?"
				reqArgs = append(reqArgs, newsupplier)
			}
			if newentrydate != nil {
				sqlr += ",storage_entrydate"
				reqValues += ",?"
				reqArgs = append(reqArgs, newentrydate)
			}
			if newexitdate != nil {
				sqlr += ",storage_exitdate"
				reqValues += ",?"
				reqArgs = append(reqArgs, newexitdate)
			}
			if newopeningdate != nil {
				sqlr += ",storage_openingdate"
				reqValues += ",?"
				reqArgs = append(reqArgs, newopeningdate)
			}
			if newexpirationdate != nil {
				sqlr += ",storage_expirationdate"
				reqValues += ",?"
				reqArgs = append(reqArgs, newexpirationdate)
			}

			sqlr += `) VALUES (` + reqValues + `)`

			globals.Log.Debug(globals.Log.WithFields(logrus.Fields{
				"newstorageCreationdate": newstorageCreationdate,
				"newcomment":             newcomment,
				"newreference":           newreference,
				"newbatchNumber":         newbatchNumber,
				"newvolumeWeight":        newvolumeWeight,
				"newbarecode":            newbarecode,
				"newtoDestroy":           newtoDestroy,
				"newperson":              newperson,
				"newproduct":             newproduct,
				"newstoreLocation":       newstoreLocation,
				"newunit":                newunit,
				"newsupplier":            newsupplier,
				"newentrydate":           newentrydate,
				"newexitdate":            newexitdate,
				"newopeningdate":         newopeningdate,
				"newexpirationdate":      newexpirationdate,
			}))

			if _, err = tx.Exec(sqlr, reqArgs...); err != nil {
				if errr := tx.Rollback(); errr != nil {
					return errr
				}
				return err
			}
		}
	}

	// committing changes
	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	// beginning new transaction
	if tx, err = db.Beginx(); err != nil {
		return err
	}

	globals.Log.Info("- updating storages qr codes (long task)")
	var sts []Storage
	var png []byte
	if err = db.Select(&sts, ` SELECT storage_id
        FROM storage`); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	// beginning new transaction
	if tx, err = db.Beginx(); err != nil {
		return err
	}
	for _, s := range sts {

		// generating qrcode
		newqrcode := globals.ProxyURL + globals.ProxyPath + "v/storages?storage=" + strconv.FormatInt(s.StorageID.Int64, 10)
		globals.Log.Debug("  " + strconv.FormatInt(s.StorageID.Int64, 10) + " " + newqrcode)

		if png, err = qrcode.Encode(newqrcode, qrcode.Medium, 512); err != nil {
			return err
		}
		sqlr = `UPDATE storage
            SET storage_qrcode = ?
            WHERE storage_id = ?`
		if _, err = tx.Exec(sqlr, png, s.StorageID); err != nil {
			globals.Log.Error("error updating storage qrcode")
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
	}

	globals.Log.Info("- updating store locations full path")
	var sls []StoreLocation
	if err = db.Select(&sls, ` SELECT s.storelocation_id AS "storelocation_id", 
        s.storelocation_name AS "storelocation_name", 
        s.storelocation_canstore, 
        s.storelocation_color,
        storelocation.storelocation_id AS "storelocation.storelocation_id",
        storelocation.storelocation_name AS "storelocation.storelocation_name"
        FROM storelocation AS s
        LEFT JOIN storelocation on s.storelocation = storelocation.storelocation_id`); err != nil {
		return err
	}

	// // beginning new transaction
	// if tx, err = db.Beginx(); err != nil {
	// 	return err
	// }
	for _, sl := range sls {
		globals.Log.Debug("  " + sl.StoreLocationName.String)
		sl.StoreLocationFullPath = db.buildFullPath(sl, tx)
		sqlr = `UPDATE storelocation SET storelocation_fullpath = ? WHERE storelocation_id = ?`
		if _, err = tx.Exec(sqlr, sl.StoreLocationFullPath, sl.StoreLocationID.Int64); err != nil {
			if errr := tx.Rollback(); errr != nil {
				return errr
			}
			return err
		}
	}

	// committing changes
	if err = tx.Commit(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	return nil
}
