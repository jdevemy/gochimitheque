package datastores

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx" // register sqlite3 driver
	"github.com/sirupsen/logrus"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/tbellembois/gochimitheque/logger"
	. "github.com/tbellembois/gochimitheque/models"
)

// ToogleStorageBorrowing toogles the borrowing b
func (db *SQLiteDataStore) ToogleStorageBorrowing(s Storage) error {
	var (
		sqlr  string
		count int
		err   error
	)

	sqlr = `SELECT COUNT(borrowing_id) FROM borrowing WHERE storage = ?`
	if err = db.Get(&count, sqlr, s.StorageID.Int64); err != nil {
		return err
	}

	if count == 0 {
		sqlr = `INSERT into borrowing(person, storage, borrower, borrowing_comment) VALUES (?, ?, ?, ?)`
		if _, err = db.Exec(sqlr, s.Borrowing.Person.PersonID, s.StorageID.Int64, s.Borrowing.Borrower.PersonID, s.Borrowing.BorrowingComment); err != nil {
			return err
		}
	} else {
		sqlr = `DELETE from borrowing WHERE storage = ?`
		if _, err = db.Exec(sqlr, s.StorageID.Int64); err != nil {
			return err
		}
	}

	return nil
}

// GetStoragesUnits return the units matching the search criteria
func (db *SQLiteDataStore) GetStoragesUnits(p DbselectparamUnit) ([]Unit, int, error) {
	var (
		units                              []Unit
		count                              int
		precreq, presreq, comreq, postsreq strings.Builder
		cnstmt                             *sqlx.NamedStmt
		snstmt                             *sqlx.NamedStmt
		err                                error
	)

	precreq.WriteString(" SELECT count(DISTINCT unit.unit_id)")
	presreq.WriteString(" SELECT unit_id, unit_label, unit_type")

	comreq.WriteString(" FROM unit")
	comreq.WriteString(" WHERE unit_label LIKE :search")

	if p.GetUnitType() != "" {
		comreq.WriteString(" AND unit_type=:unit_type")
	}

	postsreq.WriteString(" ORDER BY unit.unit_type, unit_id  " + p.GetOrder())

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
		"search":    p.GetSearch(),
		"order":     p.GetOrder(),
		"limit":     p.GetLimit(),
		"offset":    p.GetOffset(),
		"unit_type": p.GetUnitType(),
	}

	// select
	if err = snstmt.Select(&units, m); err != nil {
		return nil, 0, err
	}
	// count
	if err = cnstmt.Get(&count, m); err != nil {
		return nil, 0, err
	}

	logger.Log.WithFields(logrus.Fields{"units": units}).Debug("GetStoragesUnits")
	return units, count, nil
}

// GetStoragesSuppliers return the suppliers matching the search criteria
func (db *SQLiteDataStore) GetStoragesSuppliers(p Dbselectparam) ([]Supplier, int, error) {
	var (
		suppliers                          []Supplier
		count                              int
		exactSearch                        string
		precreq, presreq, comreq, postsreq strings.Builder
		cnstmt                             *sqlx.NamedStmt
		snstmt                             *sqlx.NamedStmt
		err                                error
	)

	exactSearch = p.GetSearch()
	exactSearch = strings.TrimPrefix(exactSearch, "%")
	exactSearch = strings.TrimSuffix(exactSearch, "%")

	precreq.WriteString(" SELECT count(DISTINCT supplier.supplier_id)")
	presreq.WriteString(" SELECT supplier_id, supplier_label")

	comreq.WriteString(" FROM supplier")
	comreq.WriteString(" WHERE supplier_label LIKE :search")
	postsreq.WriteString(" ORDER BY INSTR(supplier_label, \"" + exactSearch + "\") ASC, supplier_label ASC")

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
		"search": p.GetSearch(),
		"order":  p.GetOrder(),
		"limit":  p.GetLimit(),
		"offset": p.GetOffset(),
	}

	// select
	if err = snstmt.Select(&suppliers, m); err != nil {
		return nil, 0, err
	}
	// count
	if err = cnstmt.Get(&count, m); err != nil {
		return nil, 0, err
	}

	// setting the C attribute for formula matching exactly the search
	var supplier Supplier

	r := db.QueryRowx(`SELECT supplier_id, supplier_label FROM supplier WHERE supplier_label == ?`, exactSearch)
	if err = r.StructScan(&supplier); err != nil && err != sql.ErrNoRows {
		return nil, 0, err
	}
	for i, s := range suppliers {
		if s.SupplierID == supplier.SupplierID {
			suppliers[i].C = 1
		}
	}

	logger.Log.WithFields(logrus.Fields{"suppliers": suppliers}).Debug("GetStoragesSuppliers")
	return suppliers, count, nil
}

// GetStorages returns the storages matching the request parameters p
// Only storages that the logged user can see are returned given his permissions
// and membership
func (db *SQLiteDataStore) GetStorages(p DbselectparamStorage) ([]Storage, int, error) {
	var (
		storages                                  []Storage
		count                                     int
		precreq, presreq, comreq, postsreq, reqhc strings.Builder
		cnstmt                                    *sqlx.NamedStmt
		snstmt                                    *sqlx.NamedStmt
		err                                       error
		isadmin                                   bool
	)
	logger.Log.WithFields(logrus.Fields{"p": p}).Debug("GetStorages")

	if strings.HasPrefix(p.GetOrderBy(), "storage_") {
		p.SetOrderBy("s." + p.GetOrderBy())
	}

	// is the user an admin?
	if isadmin, err = db.IsPersonAdmin(p.GetLoggedPersonID()); err != nil {
		return nil, 0, err
	}

	// pre request: select or count
	precreq.WriteString(" SELECT count(DISTINCT s.storage_id)")
	presreq.WriteString(` SELECT s.storage_id AS "storage_id",
		s.storage_entrydate,
		s.storage_exitdate,
		s.storage_openingdate,
		s.storage_expirationdate,
		s.storage_reference,
		s.storage_batchnumber,
		s.storage_todestroy,
		s.storage_creationdate,
		s.storage_modificationdate,
		s.storage_quantity,
		s.storage_barecode,
		s.storage_qrcode,
		s.storage_comment,
		s.storage_archive,
		s.storage_concentration,
		s.storage_number_of_carton,
		s.storage_number_of_bag,
		s.storage_number_of_unit,
		storage.storage_id AS "storage.storage_id",
		uq.unit_id AS "unit_quantity.unit_id",
		uq.unit_label AS "unit_quantity.unit_label",
		uc.unit_id AS "unit_concentration.unit_id",
		uc.unit_label AS "unit_concentration.unit_label",
		supplier.supplier_id AS "supplier.supplier_id",
		supplier.supplier_label AS "supplier.supplier_label",
		person.person_id AS "person.person_id", 
		person.person_email AS "person.person_email", 
		product.product_id AS "product.product_id",
		product.product_specificity AS "product.product_specificity",
		product.product_number_per_carton AS "product.product_number_per_carton",
		product.product_number_per_bag AS "product.product_number_per_bag",
        producerref.producerref_id AS "product.producerref.producerref_id",
		name.name_id AS "product.name.name_id",
		name.name_label AS "product.name.name_label",
		casnumber.casnumber_id AS "product.casnumber.casnumber_id",
		casnumber.casnumber_label AS "product.casnumber.casnumber_label",
		borrowing.borrowing_id AS "borrowing.borrowing_id",
		borrowing.borrowing_comment AS "borrowing.borrowing_comment",
		storelocation.storelocation_id AS "storelocation.storelocation_id",
		storelocation.storelocation_name AS "storelocation.storelocation_name",
		storelocation.storelocation_color AS "storelocation.storelocation_color",
		storelocation.storelocation_fullpath AS "storelocation.storelocation_fullpath",
		entity.entity_id AS "storelocation.entity.entity_id"
		`)

	// borrower.person_id AS "borrowing.borrower.person_id",
	// borrower.person_email AS "borrowing.borrower.person_email",

	// common parts
	comreq.WriteString(" FROM storage as s")
	// get storage history parent
	comreq.WriteString(" LEFT JOIN storage ON s.storage = storage.storage_id")
	// get product
	comreq.WriteString(" JOIN product ON s.product = product.product_id")
	// get producerref
	if p.GetProducerRef() != -1 {
		comreq.WriteString(" JOIN producerref ON product.producerref = :producerref")
	} else {
		comreq.WriteString(" LEFT JOIN producerref ON product.producerref = producerref.producerref_id")
	}
	// get name
	comreq.WriteString(" JOIN name ON product.name = name.name_id")
	// get signal word
	comreq.WriteString(" LEFT JOIN signalword ON product.signalword = signalword.signalword_id")
	// get person
	comreq.WriteString(" JOIN person ON s.person = person.person_id")
	// get store location
	comreq.WriteString(" JOIN storelocation ON s.storelocation = storelocation.storelocation_id")
	// get entity
	comreq.WriteString(" JOIN entity ON storelocation.entity = entity.entity_id")
	// get unit quantity
	comreq.WriteString(" LEFT JOIN unit uq ON s.unit_quantity = uq.unit_id")
	// get unit concentration
	comreq.WriteString(" LEFT JOIN unit uc ON s.unit_concentration = uc.unit_id")
	// get supplier
	comreq.WriteString(" LEFT JOIN supplier ON s.supplier = supplier.supplier_id")
	// get borrowings
	if p.GetBorrowing() {
		comreq.WriteString(" JOIN borrowing ON borrowing.storage = s.storage_id AND borrowing.borrower = :personid")
	} else {
		comreq.WriteString(" LEFT JOIN borrowing ON s.storage_id = borrowing.storage")
	}
	//comreq.WriteString(" LEFT JOIN person AS borrower ON borrowing.borrower = borrower.person_id")

	// get name
	//comreq.WriteString(" JOIN name ON product.name = name.name_id")
	// get CMR
	if p.GetCasNumberCmr() {
		comreq.WriteString(" JOIN casnumber ON product.casnumber = casnumber.casnumber_id AND casnumber.casnumber_cmr IS NOT NULL")
	} else {
		// get casnumber
		comreq.WriteString(" LEFT JOIN casnumber ON product.casnumber = casnumber.casnumber_id")
	}
	// get empirical formula
	comreq.WriteString(" LEFT JOIN empiricalformula ON product.empiricalformula = empiricalformula.empiricalformula_id")
	// get symbols
	if len(p.GetSymbols()) != 0 {
		comreq.WriteString(" JOIN productsymbols AS ps ON ps.productsymbols_product_id = product.product_id")
	}
	// get hazardstatements
	if len(p.GetHazardStatements()) != 0 {
		comreq.WriteString(" JOIN producthazardstatements AS phs ON phs.producthazardstatements_product_id = product.product_id")
	}
	// get precautionarystatements
	if len(p.GetPrecautionaryStatements()) != 0 {
		comreq.WriteString(" JOIN productprecautionarystatements AS pps ON pps.productprecautionarystatements_product_id = product.product_id")
	}
	// get bookmarks
	if p.GetBookmark() {
		comreq.WriteString(" JOIN bookmark AS b ON b.product = product.product_id AND b.person = :personid")
	}

	// filter by entities
	if !isadmin {
		comreq.WriteString(` JOIN personentities ON (personentities_entity_id = storelocation.entity AND personentities_person_id = :personid)`)
	}

	// filter by permissions
	comreq.WriteString(` JOIN permission AS perm, entity as e ON
	perm.person = :personid and (perm.permission_item_name in ("all", "storages")) and (perm.permission_perm_name in ("all", "r", "w")) and (perm.permission_entity_id in (-1, e.entity_id))
	`)
	comreq.WriteString(" WHERE 1")
	if len(p.GetIds()) > 0 {
		comreq.WriteString(" AND s.storage_id in (")

		for _, id := range p.GetIds() {
			comreq.WriteString(fmt.Sprintf("%d,", id))
		}
		// to complete the last comma
		comreq.WriteString("-1")
		comreq.WriteString(" )")

	}
	if p.GetStorageToDestroy() {
		comreq.WriteString(" AND s.storage_todestroy = true")
	}
	if p.GetProduct() != -1 {
		comreq.WriteString(" AND product.product_id = :product")
	}
	if p.GetEntity() != -1 {
		comreq.WriteString(" AND entity.entity_id = :entity")
	}
	if p.GetStorelocation() != -1 {
		comreq.WriteString(" AND storelocation.storelocation_id = :storelocation")
	}
	if p.GetStorage() != -1 {
		if p.GetHistory() {
			comreq.WriteString(" AND (s.storage = :storage OR s.storage_id = :storage)")
		} else {
			comreq.WriteString(" AND (s.storage_id = :storage")
			// getting storages with identical barecode
			comreq.WriteString(" OR (s.storage_barecode = (SELECT storage_barecode FROM storage WHERE storage_id = :storage)))")
		}
	}
	if !p.GetHistory() {
		comreq.WriteString(" AND s.storage IS NULL")
	}
	if p.GetStorageArchive() {
		comreq.WriteString(" AND s.storage_archive = true")
	} else {
		comreq.WriteString(" AND s.storage_archive = false")
	}

	// search form parameters
	if p.GetName() != -1 {
		comreq.WriteString(" AND name.name_id = :name")
	}
	if p.GetCasNumber() != -1 {
		comreq.WriteString(" AND casnumber.casnumber_id = :casnumber")
	}
	if p.GetEmpiricalFormula() != -1 {
		comreq.WriteString(" AND empiricalformula.empiricalformula_id = :empiricalformula")
	}
	if p.GetStorageBarecode() != "" {
		comreq.WriteString(" AND s.storage_barecode LIKE :storage_barecode")
	}
	if p.GetStorageBatchNumber() != "" {
		comreq.WriteString(" AND s.storage_batchnumber LIKE :storage_batchnumber")
	}
	if p.GetCustomNamePartOf() != "" {
		comreq.WriteString(" AND name.name_label LIKE :custom_name_part_of")
	}
	if len(p.GetSymbols()) != 0 {
		comreq.WriteString(" AND ps.productsymbols_symbol_id IN (")
		for _, s := range p.GetSymbols() {
			comreq.WriteString(fmt.Sprintf("%d,", s))
		}
		// to complete the last comma
		comreq.WriteString("-1")
		comreq.WriteString(" )")
	}
	if len(p.GetHazardStatements()) != 0 {
		comreq.WriteString(" AND phs.producthazardstatements_hazardstatement_id IN (")
		for _, s := range p.GetHazardStatements() {
			comreq.WriteString(fmt.Sprintf("%d,", s))
		}
		// to complete the last comma
		comreq.WriteString("-1")
		comreq.WriteString(" )")
	}
	if len(p.GetPrecautionaryStatements()) != 0 {
		comreq.WriteString(" AND pps.productprecautionarystatements_precautionarystatement_id IN (")
		for _, s := range p.GetPrecautionaryStatements() {
			comreq.WriteString(fmt.Sprintf("%d,", s))
		}
		// to complete the last comma
		comreq.WriteString("-1")
		comreq.WriteString(" )")
	}
	if p.GetSignalWord() != -1 {
		comreq.WriteString(" AND signalword.signalword_id = :signalword")
	}

	// show bio/chem/consu
	if !p.GetShowChem() && !p.GetShowBio() && p.GetShowConsu() {
		comreq.WriteString(" AND (product_number_per_carton IS NOT NULL AND product_number_per_carton != 0)")
	} else if !p.GetShowChem() && p.GetShowBio() && !p.GetShowConsu() {
		comreq.WriteString(" AND producerref IS NOT NULL")
		comreq.WriteString(" AND (product_number_per_carton IS NULL OR product_number_per_carton == 0)")
	} else if !p.GetShowChem() && p.GetShowBio() && p.GetShowConsu() {
		comreq.WriteString(" AND ((product_number_per_carton IS NOT NULL AND product_number_per_carton != 0)")
		comreq.WriteString(" OR producerref IS NOT NULL)")
	} else if p.GetShowChem() && !p.GetShowBio() && !p.GetShowConsu() {
		comreq.WriteString(" AND producerref IS NULL")
		comreq.WriteString(" AND (product_number_per_carton IS NULL OR product_number_per_carton == 0)")
	} else if p.GetShowChem() && !p.GetShowBio() && p.GetShowConsu() {
		comreq.WriteString(" AND (producerref IS NULL")
		comreq.WriteString(" OR (product_number_per_carton IS NOT NULL AND product_number_per_carton != 0))")
	} else if p.GetShowChem() && p.GetShowBio() && !p.GetShowConsu() {
		comreq.WriteString(" AND (product_number_per_carton IS NULL OR product_number_per_carton == 0)")
	}

	// post select request
	postsreq.WriteString(" GROUP BY s.storage_id")
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
		"ids":                 p.GetIds(),
		"search":              p.GetSearch(),
		"personid":            p.GetLoggedPersonID(),
		"order":               p.GetOrder(),
		"limit":               p.GetLimit(),
		"offset":              p.GetOffset(),
		"entity":              p.GetEntity(),
		"product":             p.GetProduct(),
		"storelocation":       p.GetStorelocation(),
		"storage":             p.GetStorage(),
		"name":                p.GetName(),
		"casnumber":           p.GetCasNumber(),
		"empiricalformula":    p.GetEmpiricalFormula(),
		"storage_barecode":    p.GetStorageBarecode(),
		"storage_batchnumber": p.GetStorageBatchNumber(),
		"custom_name_part_of": "%" + p.GetCustomNamePartOf() + "%",
		"signalword":          p.GetSignalWord(),
		"producerref":         p.GetProducerRef(),
	}

	// select
	if err = snstmt.Select(&storages, m); err != nil {
		return nil, 0, err
	}
	// count
	if err = cnstmt.Get(&count, m); err != nil {
		return nil, 0, err
	}

	//
	// getting number of history for each storage
	//
	for i, st := range storages {
		// getting the total storage count
		//logger.Log.Debug(st)
		reqhc.Reset()
		reqhc.WriteString("SELECT count(DISTINCT storage_id) from storage WHERE storage.storage = ?")
		if err = db.Get(&storages[i].StorageHC, reqhc.String(), st.StorageID); err != nil {
			return nil, 0, err
		}
	}

	//
	// getting borrower for each storage
	//
	for i, st := range storages {
		reqhc.Reset()
		reqhc.WriteString(`SELECT borrowing_id, 
		borrowing_comment, 
		person.person_email AS "borrower.person_email" 
		from borrowing 
		JOIN person 
		ON borrowing.borrower = person.person_id 
		WHERE borrowing.storage = ?`)
		var borrowing Borrowing
		if err = db.Get(&borrowing, reqhc.String(), st.StorageID); err != nil && err != sql.ErrNoRows {
			return nil, 0, err
		}
		storages[i].Borrowing = &borrowing
	}

	return storages, count, nil
}

// GetOtherStorages returns the entity manager(s) email of the entities
// storing the product with the id passed in the request parameters p
func (db *SQLiteDataStore) GetOtherStorages(p DbselectparamStorage) ([]Entity, int, error) {
	var (
		entities                           []Entity
		count                              int
		precreq, presreq, comreq, postsreq strings.Builder
		cnstmt                             *sqlx.NamedStmt
		snstmt                             *sqlx.NamedStmt
		err                                error
	)
	logger.Log.WithFields(logrus.Fields{"p": p}).Debug("GetOtherStorages")

	// pre request: select or count
	precreq.WriteString(" SELECT count(DISTINCT e.entity_id)")
	presreq.WriteString(` SELECT e.entity_id AS "entity_id",
	e.entity_name AS "entity_name",
	GROUP_CONCAT(DISTINCT person.person_email) AS "entity_description"
	`)

	// common parts
	comreq.WriteString(" FROM entity as e")

	// get store location
	comreq.WriteString(" JOIN storelocation ON storelocation.entity = e.entity_id")
	// get storages
	comreq.WriteString(" JOIN storage ON storage.storelocation = storelocation.storelocation_id")

	// get managers
	comreq.WriteString(" JOIN entitypeople ON e.entity_id = entitypeople.entitypeople_entity_id")
	comreq.WriteString(" JOIN person ON entitypeople.entitypeople_person_id = person.person_id")

	comreq.WriteString(" WHERE 1")
	if p.GetProduct() != -1 {
		comreq.WriteString(" AND storage.product = :product")
	}

	// post select request
	postsreq.WriteString(" GROUP BY e.entity_id")

	// building count and select statements
	if cnstmt, err = db.PrepareNamed(precreq.String() + comreq.String()); err != nil {
		return nil, 0, err
	}
	if snstmt, err = db.PrepareNamed(presreq.String() + comreq.String() + postsreq.String()); err != nil {
		return nil, 0, err
	}

	// building argument map
	m := map[string]interface{}{
		"search":              p.GetSearch(),
		"personid":            p.GetLoggedPersonID(),
		"order":               p.GetOrder(),
		"limit":               p.GetLimit(),
		"offset":              p.GetOffset(),
		"entity":              p.GetEntity(),
		"product":             p.GetProduct(),
		"storelocation":       p.GetStorelocation(),
		"storage":             p.GetStorage(),
		"name":                p.GetName(),
		"casnumber":           p.GetCasNumber(),
		"empiricalformula":    p.GetEmpiricalFormula(),
		"storage_barecode":    p.GetStorageBarecode(),
		"custom_name_part_of": "%" + p.GetCustomNamePartOf() + "%",
		"signalword":          p.GetSignalWord(),
	}

	// select
	if err = snstmt.Select(&entities, m); err != nil {
		return nil, 0, err
	}
	// count
	if err = cnstmt.Get(&count, m); err != nil {
		return nil, 0, err
	}

	return entities, count, nil
}

// GetStorage returns the storage with id "id"
func (db *SQLiteDataStore) GetStorage(id int) (Storage, error) {
	var (
		storage Storage
		sqlr    string
		err     error
	)
	logger.Log.WithFields(logrus.Fields{"id": id}).Debug("GetStorage")

	sqlr = `SELECT storage.storage_id,
	storage.storage_entrydate,
	storage.storage_exitdate,
	storage.storage_openingdate,
	storage.storage_expirationdate,
	storage.storage_reference,
	storage.storage_batchnumber,
	storage.storage_todestroy,
	storage.storage_creationdate,
	storage.storage_modificationdate,
	storage.storage_quantity,
	storage.storage_barecode,
	storage.storage_qrcode,
	storage.storage_comment,
	storage.storage_archive,
	storage.storage_number_of_carton,
	storage.storage_number_of_bag,
	storage.storage_number_of_unit,
	uq.unit_id AS "unit_quantity.unit_id",
	uq.unit_label AS "unit_quantity.unit_label",
	uc.unit_id AS "unit_concentration.unit_id",
	uc.unit_label AS "unit_concentration.unit_label",
	supplier.supplier_id AS "supplier.supplier_id",
	supplier.supplier_label AS "supplier.supplier_label",
	person.person_id AS "person.person_id",
	person.person_email AS "person.person_email",
	name.name_id AS "product.name.name_id",
	name.name_label AS "product.name.name_label",
	product.product_id AS "product.product_id",
	product.product_number_per_carton AS "product.product_number_per_carton",
	producerref.producerref_id AS "product.producerref.producerref_id",
	casnumber.casnumber_id AS "product.casnumber.casnumber_id",
	casnumber.casnumber_label AS "product.casnumber.casnumber_label",
	storelocation.storelocation_id AS "storelocation.storelocation_id",
	storelocation.storelocation_name AS "storelocation.storelocation_name",
	storelocation.storelocation_color AS "storelocation.storelocation_color",
	storelocation.storelocation_fullpath AS "storelocation.storelocation_fullpath",
	entity.entity_id AS "storelocation.entity.entity_id"
	FROM storage
	JOIN storelocation ON storage.storelocation = storelocation.storelocation_id
	JOIN entity ON storelocation.entity = entity.entity_id
	LEFT JOIN unit uq ON storage.unit_quantity = uq.unit_id
	LEFT JOIN unit uc ON storage.unit_concentration = uc.unit_id
	LEFT JOIN supplier ON storage.supplier = supplier.supplier_id
	JOIN person ON storage.person = person.person_id
	JOIN product ON storage.product = product.product_id
	LEFT JOIN producerref ON product.producerref = producerref.producerref_id
	LEFT JOIN casnumber ON product.casnumber = casnumber.casnumber_id
	JOIN name ON product.name = name.name_id
	WHERE storage.storage_id = ?`
	if err = db.Get(&storage, sqlr, id); err != nil {
		return Storage{}, err
	}
	logger.Log.WithFields(logrus.Fields{"ID": id, "storage": storage}).Debug("GetStorage")
	return storage, nil
}

// GetStorageEntity returns the entity of the storage with id "id"
func (db *SQLiteDataStore) GetStorageEntity(id int) (Entity, error) {
	var (
		entity Entity
		sqlr   string
		err    error
	)

	sqlr = `SELECT 
	entity.entity_id AS "entity_id",
	entity.entity_name AS "entity_name"
	FROM storage
	JOIN storelocation ON storage.storelocation = storelocation.storelocation_id
	JOIN entity ON storelocation.entity = entity.entity_id
	WHERE storage.storage_id = ?`
	if err = db.Get(&entity, sqlr, id); err != nil {
		return Entity{}, err
	}
	logger.Log.WithFields(logrus.Fields{"ID": id, "entity": entity}).Debug("GetStorageEntity")
	return entity, nil
}

// DeleteStorage deletes the storages with the given id
func (db *SQLiteDataStore) DeleteStorage(id int) error {

	var (
		sqlr string
		err  error
	)
	sqlr = `DELETE FROM storage 
	WHERE storage_id = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}
	return nil
}

// ArchiveStorage archives the storages with the given id
func (db *SQLiteDataStore) ArchiveStorage(id int) error {

	var (
		sqlr string
		err  error
	)
	sqlr = `UPDATE storage SET storage_archive = true 
	WHERE storage_id = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}
	sqlr = `UPDATE storage SET storage_archive = true 
	WHERE storage.storage = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}

	return nil
}

// RestoreStorage restores (unarchive) the storages with the given id
func (db *SQLiteDataStore) RestoreStorage(id int) error {

	var (
		sqlr string
		err  error
	)
	sqlr = `UPDATE storage SET storage_archive = false 
	WHERE storage_id = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}
	sqlr = `UPDATE storage SET storage_archive = false 
	WHERE storage.storage = ?`
	if _, err = db.Exec(sqlr, id); err != nil {
		return err
	}

	return nil
}

// GenerateAndUpdateStorageBarecode generate and set a barecode for the storage s
// the barecode is [prefix]major.minor
// with
// prefix: extracted from the storelocation name [prefix]storelocation_name, or ""
// major: unique uid identical for the differents storages of the same product in an entity
// minor: incremental number for the differents storages of the same product in an entity
// func (db *SQLiteDataStore) GenerateAndUpdateStorageBarecode(s *Storage) error {

// 	var (
// 		err      error
// 		m        []string
// 		png      []byte
// 		lastbc   []string
// 		prefix   string
// 		major    string
// 		minor    string
// 		iminor   int
// 		barecode string
// 	)

// 	// defaults
// 	major = strconv.Itoa(s.ProductID)
// 	minor = "0"

// 	logger.Log.WithFields(logrus.Fields{"s": s}).Debug("GenerateAndUpdateStorageBarecode")

// 	//
// 	// prefix
// 	//
// 	// regex to detect store locations names starting with [a-zA-Z] to build barecode prefixes
// 	slr := regexp.MustCompile(`^\[(?P<groupone>[a-zA-Z]{1})\].*$`)
// 	// finding group names
// 	n := slr.SubexpNames()
// 	// finding matches
// 	ms := slr.FindAllStringSubmatch(s.StoreLocationName.String, -1)
// 	// then building a map of matches
// 	md := map[string]string{}
// 	if len(ms) != 0 {
// 		m = ms[0]
// 		for i, j := range m {
// 			md[n[i]] = j
// 		}
// 	}
// 	if len(md) > 0 {
// 		prefix = md["groupone"]
// 	}

// 	//
// 	// major
// 	//
// 	// getting the last storage barecode
// 	// for the same product
// 	// in the same entity
// 	sqlr := `SELECT storage_barecode FROM storage
// 	WHERE NOT storage_barecode IS NULL AND product = ? AND storelocation = ?
// 	ORDER BY storage_barecode DESC`
// 	if err = db.Select(&lastbc, sqlr, s.ProductID, s.StoreLocationID); err != nil && err != sql.ErrNoRows {
// 		logger.Log.Error("error getting the last storage barecode")
// 		return err
// 	}
// 	logger.Log.WithFields(logrus.Fields{"lastbc": lastbc}).Debug("GenerateAndUpdateStorageBarecode")

// 	// regex to extract the major and minor from a barecode
// 	majorr := regexp.MustCompile(`^[a-zA-Z]{0,1}(?P<groupone>[0-9]+)\.(?P<grouptwo>[0-9]+)$`)
// 	// finding group names
// 	n = majorr.SubexpNames()
// 	for _, bc := range lastbc {

// 		// finding matches
// 		ms = majorr.FindAllStringSubmatch(bc, -1)
// 		if ms != nil {

// 			// then building a map of matches
// 			md = map[string]string{}
// 			if len(ms) != 0 {
// 				m = ms[0]
// 				for i, j := range m {
// 					md[n[i]] = j
// 				}
// 			}
// 			major = md["groupone"]
// 			minor = md["grouptwo"]
// 			logger.Log.WithFields(logrus.Fields{"major": major, "minor": minor}).Debug("GenerateAndUpdateStorageBarecode")

// 			break

// 		}
// 	}

// 	if iminor, err = strconv.Atoi(minor); err != nil {
// 		return err
// 	}
// 	iminor++
// 	minor = strconv.Itoa(iminor)
// 	barecode = prefix + major + "." + minor
// 	logger.Log.WithFields(logrus.Fields{"barecode": barecode}).Debug("GenerateAndUpdateStorageBarecode")

// 	// sqlr := `UPDATE storage
// 	// SET storage_barecode = '` + prefix + `' || storage.product || '.' || (select count(*) from storage join storelocation on storage.storelocation = storelocation.storelocation_id join entity on storelocation.entity = entity.entity_id where storage.product = ? and entity_id = ?)
// 	// WHERE storage_id = ?`
// 	sqlr = `UPDATE storage
// 	SET storage_barecode = :barecode
// 	WHERE storage_id = :storage`
// 	if _, err = db.NamedExec(sqlr, map[string]interface{}{
// 		"barecode": barecode,
// 		"storage":  s.StorageID.Int64,
// 	}); err != nil {
// 		logger.Log.Error("error updating storage barecode")
// 		return err
// 	}

// 	//
// 	// qrcode
// 	//
// 	qr := strconv.FormatInt(s.StorageID.Int64, 10)
// 	if png, err = qrcode.Encode(qr, qrcode.Medium, 512); err != nil {
// 		return err
// 	}
// 	sqlr = `UPDATE storage
// 	SET storage_qrcode = ?
// 	WHERE storage_id = ?`
// 	if _, err = db.Exec(sqlr, png, s.StorageID.Int64); err != nil {
// 		logger.Log.Error("error updating storage qr code")
// 		return err
// 	}

// 	return nil
// }

// CreateStorage creates a new storage
func (db *SQLiteDataStore) CreateStorage(s Storage, itemNumber int) (int, error) {

	var (
		lastid       int64
		tx           *sql.Tx
		sqlr         string
		res          sql.Result
		sqla         []interface{}
		ibuilder     sq.InsertBuilder
		err          error
		prefix       string
		major, minor string
	)

	// Default major.
	major = strconv.Itoa(s.ProductID)

	if tx, err = db.Begin(); err != nil {
		return 0, err
	}

	// Generating barecode if empty.
	if !(s.StorageBarecode.Valid) || s.StorageBarecode.String == "" {

		//
		// Getting the barecode prefix from the storelocation name.
		//
		// regex to detect store locations names starting with [_a-zA-Z] to build barecode prefixes
		prefixRegex := regexp.MustCompile(`^\[(?P<groupone>[_a-zA-Z]{1,5})\].*$`)
		groupNames := prefixRegex.SubexpNames()
		matches := prefixRegex.FindAllStringSubmatch(s.StoreLocationName.String, -1)
		// Building a map of matches.
		matchesMap := map[string]string{}
		if len(matches) != 0 {
			for i, j := range matches[0] {
				matchesMap[groupNames[i]] = j
			}
		}

		if len(matchesMap) > 0 {
			prefix = matchesMap["groupone"]
		} else {
			prefix = "_"
		}

		//
		// Getting the storage barecodes matching the regex
		// for the same product in the same entity.
		//
		sqlr := `SELECT storage_barecode FROM storage 
		JOIN storelocation on storage.storelocation = storelocation.storelocation_id 
		WHERE product = ? AND storelocation.entity = ? AND regexp('^[_a-zA-Z]{0,5}[0-9]+\.[0-9]+$', '' || storage_barecode || '') = true
		ORDER BY storage_barecode desc`
		var rows *sql.Rows
		if rows, err = tx.Query(sqlr, s.ProductID, s.EntityID); err != nil && err != sql.ErrNoRows {
			if errr := tx.Rollback(); errr != nil {
				return 0, errr
			}
			return 0, err
		}

		var (
			count    = 0
			newMinor = 0
		)
		for rows.Next() {

			var barecode string
			if err = rows.Scan(&barecode); err != nil && err != sql.ErrNoRows {
				if errr := tx.Rollback(); errr != nil {
					return 0, errr
				}
				return 0, err
			}

			majorRegex := regexp.MustCompile(`^[_a-zA-Z]{0,5}(?P<groupone>[0-9]+)\.(?P<grouptwo>[0-9]+)$`)
			groupNames = majorRegex.SubexpNames()
			matches = majorRegex.FindAllStringSubmatch(barecode, -1)
			// Building a map of matches.
			matchesMap = map[string]string{}
			if len(matches) != 0 {
				for i, j := range matches[0] {
					matchesMap[groupNames[i]] = j
				}
			}

			if count == 0 {
				// All of the major number are the same.
				// Extracting it ones.
				major = matchesMap["groupone"]
			}
			minor = matchesMap["grouptwo"]
			var iminor int
			if iminor, err = strconv.Atoi(minor); err != nil {
				return 0, err
			}

			if iminor > newMinor {
				newMinor = iminor
			}

			count++

		}

		if (!s.StorageIdenticalBarecode.Valid || !s.StorageIdenticalBarecode.Bool) || (s.StorageIdenticalBarecode.Valid && s.StorageIdenticalBarecode.Bool && itemNumber == 1) {
			newMinor++
		}
		minor = strconv.Itoa(newMinor)
		s.StorageBarecode.String = prefix + major + "." + minor
		s.StorageBarecode.Valid = true
		logger.Log.WithFields(logrus.Fields{"s.StorageBarecode.String": s.StorageBarecode.String}).Debug("CreateStorage")

	}

	// if SupplierID = -1 then it is a new supplier
	if v, err := s.Supplier.SupplierID.Value(); s.Supplier.SupplierID.Valid && err == nil && v.(int64) == -1 {
		sqlr = `INSERT INTO supplier (supplier_label) VALUES (?)`
		if res, err = tx.Exec(sqlr, s.Supplier.SupplierLabel); err != nil {
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
		// updating the storage SupplierId (SupplierLabel already set)
		s.Supplier.SupplierID = sql.NullInt64{Valid: true, Int64: lastid}
	}
	if err != nil {
		logger.Log.Error("supplier error - " + err.Error())
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, err
	}

	// finally updating the storage
	m := make(map[string]interface{})
	if s.StorageComment.Valid {
		m["storage_comment"] = s.StorageComment.String
	}
	if s.StorageQuantity.Valid {
		m["storage_quantity"] = s.StorageQuantity.Float64
	}
	if s.StorageBarecode.Valid {
		m["storage_barecode"] = s.StorageBarecode.String
	}
	if s.UnitQuantity.UnitID.Valid {
		m["unit_quantity"] = s.UnitQuantity.UnitID.Int64
	}
	if s.SupplierID.Valid {
		m["supplier"] = s.SupplierID.Int64
	}
	if s.StorageEntryDate.Valid {
		m["storage_entrydate"] = s.StorageEntryDate.Time
	}
	if s.StorageExitDate.Valid {
		m["storage_exitdate"] = s.StorageExitDate.Time
	}
	if s.StorageOpeningDate.Valid {
		m["storage_openingdate"] = s.StorageOpeningDate.Time
	}
	if s.StorageExpirationDate.Valid {
		m["storage_expirationdate"] = s.StorageExpirationDate.Time
	}
	if s.StorageReference.Valid {
		m["storage_reference"] = s.StorageReference.String
	}
	if s.StorageBatchNumber.Valid {
		m["storage_batchnumber"] = s.StorageBatchNumber.String
	}
	if s.StorageToDestroy.Valid {
		m["storage_todestroy"] = s.StorageToDestroy.Bool
	}
	if s.StorageConcentration.Valid {
		m["storage_concentration"] = int(s.StorageConcentration.Int64)
	}
	if s.StorageNumberOfBag.Valid {
		m["storage_number_of_bag"] = int(s.StorageNumberOfBag.Int64)
	}
	if s.StorageNumberOfCarton.Valid {
		m["storage_number_of_carton"] = int(s.StorageNumberOfCarton.Int64)
	}
	if s.StorageNumberOfUnit.Valid {
		m["storage_number_of_unit"] = int(s.StorageNumberOfUnit.Int64)
	}
	if s.UnitConcentration.UnitID.Valid {
		m["unit_concentration"] = int(s.UnitConcentration.UnitID.Int64)
	}

	m["person"] = s.PersonID
	m["storelocation"] = s.StoreLocationID.Int64
	m["product"] = s.ProductID
	m["storage_creationdate"] = s.StorageCreationDate
	m["storage_modificationdate"] = s.StorageModificationDate
	m["storage_archive"] = false

	// building column names/values
	col := make([]string, 0, len(m))
	val := make([]interface{}, 0, len(m))
	for k, v := range m {
		col = append(col, k)

		switch t := v.(type) {
		case int:
			val = append(val, t)
		case string:
			val = append(val, t)
		case bool:
			val = append(val, t)
		case int64:
			val = append(val, t)
		case float64:
			val = append(val, t)
		default:
			val = append(val, v)
		}
	}

	ibuilder = sq.Insert("storage").Columns(col...).Values(val...)
	if sqlr, sqla, err = ibuilder.ToSql(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return 0, errr
		}
		return 0, err
	}

	// logger.Log.Debug(sqlr)
	// logger.Log.Debug(sqla)

	if res, err = tx.Exec(sqlr, sqla...); err != nil {
		logger.Log.Error("storage error - " + err.Error())
		logger.Log.Error("sql:" + sqlr)
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

	//
	// qrcode
	//
	qr := strconv.FormatInt(lastid, 10)
	if s.StorageQRCode, err = qrcode.Encode(qr, qrcode.Medium, 512); err != nil {
		return 0, err
	}

	sqlr = `UPDATE storage SET storage_qrcode=? WHERE storage_id=?`
	if _, err = tx.Exec(sqlr, s.StorageQRCode, lastid); err != nil {
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

	s.StorageID = sql.NullInt64{Valid: true, Int64: lastid}
	logger.Log.WithFields(logrus.Fields{"s": s}).Debug("CreateStorage")

	return int(s.StorageID.Int64), nil
}

// UpdateStorage updates the storage s
func (db *SQLiteDataStore) UpdateStorage(s Storage) error {

	var (
		sqlr     string
		err      error
		tx       *sql.Tx
		res      sql.Result
		lastid   int64
		sqla     []interface{}
		ubuilder sq.UpdateBuilder
	)

	// beginning transaction
	if tx, err = db.Begin(); err != nil {
		return err
	}

	// create an history of the storage
	sqlr = `INSERT into storage (storage_creationdate, 
		storage_modificationdate,
		storage_entrydate, 
		storage_exitdate, 
		storage_openingdate, 
		storage_expirationdate,
		storage_comment,
		storage_reference,
		storage_batchnumber,
		storage_quantity,
		storage_barecode,
		storage_todestroy,
		storage_archive,
		storage_concentration,
		storage_number_of_unit integer,
		storage_number_of_bag integer,
		storage_number_of_carton integer,
		person,
		product,
		storelocation,
		unit_quantity,
		unit_concentration,
		supplier,
		storage) select storage_creationdate, 
				storage_modificationdate,
				storage_entrydate, 
				storage_exitdate, 
				storage_openingdate, 
				storage_expirationdate,
				storage_comment,
				storage_reference,
				storage_batchnumber,
				storage_quantity,
				storage_barecode,
				storage_todestroy,
				storage_archive,
				storage_concentration,
				storage_number_of_unit integer,
				storage_number_of_bag integer,
				storage_number_of_carton integer,
				person,
				product,
				storelocation,
				unit_quantity,
				unit_concentration,
				supplier,
				? FROM storage WHERE storage_id = ?`
	if _, err = tx.Exec(sqlr, s.StorageID, s.StorageID); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	// if SupplierID = -1 then it is a new supplier
	if v, err := s.Supplier.SupplierID.Value(); s.Supplier.SupplierID.Valid && err == nil && v.(int64) == -1 {
		sqlr = `INSERT INTO supplier (supplier_label) VALUES (?)`
		if _, err = tx.Exec(sqlr, s.Supplier.SupplierLabel); err != nil {
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
		// updating the storage SupplierId (SupplierLabel already set)
		s.Supplier.SupplierID = sql.NullInt64{Valid: true, Int64: lastid}
	}
	if err != nil {
		logger.Log.Error("supplier error - " + err.Error())
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}

	// finally updating the storage
	m := make(map[string]interface{})
	if s.StorageComment.Valid {
		m["storage_comment"] = s.StorageComment.String
	}
	if s.StorageQuantity.Valid {
		m["storage_quantity"] = s.StorageQuantity.Float64
	}
	if s.StorageBarecode.Valid {
		m["storage_barecode"] = s.StorageBarecode.String
	}
	if s.UnitQuantity.UnitID.Valid {
		m["unit_quantity"] = s.UnitQuantity.UnitID.Int64
	}
	if s.SupplierID.Valid {
		m["supplier"] = s.SupplierID.Int64
	}
	if s.StorageEntryDate.Valid {
		m["storage_entrydate"] = s.StorageEntryDate.Time
	}
	if s.StorageExitDate.Valid {
		m["storage_exitdate"] = s.StorageExitDate.Time
	}
	if s.StorageOpeningDate.Valid {
		m["storage_openingdate"] = s.StorageOpeningDate.Time
	}
	if s.StorageExpirationDate.Valid {
		m["storage_expirationdate"] = s.StorageExpirationDate.Time
	}
	if s.StorageReference.Valid {
		m["storage_reference"] = s.StorageReference.String
	}
	if s.StorageBatchNumber.Valid {
		m["storage_batchnumber"] = s.StorageBatchNumber.String
	}
	if s.StorageToDestroy.Valid {
		m["storage_todestroy"] = s.StorageToDestroy.Bool
	}
	if s.StorageConcentration.Valid {
		m["storage_concentration"] = int(s.StorageConcentration.Int64)
	}
	if s.StorageNumberOfBag.Valid {
		m["storage_number_of_bag"] = int(s.StorageNumberOfBag.Int64)
	}
	if s.StorageNumberOfCarton.Valid {
		m["storage_number_of_carton"] = int(s.StorageNumberOfCarton.Int64)
	}
	if s.StorageNumberOfUnit.Valid {
		m["storage_number_of_unit"] = int(s.StorageNumberOfUnit.Int64)
	}
	if s.UnitConcentration.UnitID.Valid {
		m["unit_concentration"] = int(s.UnitConcentration.UnitID.Int64)
	}
	m["storage_modificationdate"] = s.StorageModificationDate
	m["storage_archive"] = s.StorageArchive
	m["person"] = s.PersonID
	m["storelocation"] = s.StoreLocationID
	m["unit_quantity"] = s.UnitQuantity.UnitID
	m["supplier"] = s.SupplierID

	ubuilder = sq.Update("storage").
		SetMap(m).
		Where(sq.Eq{"storage_id": s.StorageID})
	if sqlr, sqla, err = ubuilder.ToSql(); err != nil {
		if errr := tx.Rollback(); errr != nil {
			return errr
		}
	}
	if _, err = tx.Exec(sqlr, sqla...); err != nil {
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

// UpdateAllQRCodes updates the storages QRCodes
func (db *SQLiteDataStore) UpdateAllQRCodes() error {

	var (
		err  error
		tx   *sqlx.Tx
		sts  []Storage
		png  []byte
		sqlr string
	)

	// retrieving storages
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
		newqrcode := strconv.FormatInt(s.StorageID.Int64, 10)
		logger.Log.Debug("  " + strconv.FormatInt(s.StorageID.Int64, 10) + " " + newqrcode)

		if png, err = qrcode.Encode(newqrcode, qrcode.Medium, 512); err != nil {
			return err
		}
		sqlr = `UPDATE storage
				SET storage_qrcode = ?
				WHERE storage_id = ?`
		if _, err = tx.Exec(sqlr, png, s.StorageID); err != nil {
			logger.Log.Error("error updating storage qrcode")
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
