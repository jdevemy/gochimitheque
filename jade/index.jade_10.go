// Code generated by "jade.go"; DO NOT EDIT.

package jade

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	index_10__16 = `</span></a></li></ul></div></nav><div id="accordion"><div id="list-collapse" class="collapse show" data-parent="#accordion"><header class="row"><div class="col-sm-12"><table id="table" data-toggle="table" data-striped="true" data-search="true" data-side-pagination="server" data-page-list="[10, 20, 50, 100]" data-pagination="true" data-ajax="getData" data-query-params="queryParams" data-sort-name="person_email" data-row-attributes="rowAttributes"><thead><tr><th data-field="person_id" data-sortable="true">ID</th><th data-field="person_email" data-sortable="true">email</th><th data-field="operate" data-formatter="operateFormatter" data-events="operateEvents">actions</th></tr></thead></table></div></header></div><div id="viewedit-collapse" class="collapse" data-parent="#accordion">`
	index_10__17 = `<form id="person"><input id="index" type="hidden" name="index" value=""/><input id="person_id" type="hidden" name="person_id" value=""/><div class="form-group row"><div class="col-sm-6">`
	index_10__19 = `</div></div><span>permissions</span><div class="d-flex justify-content-center form-group row" id="permissionsproducts"><div class="col-sm-6"><div class="iconlabel text-right">products</div></div><div class="col-sm-6"><div class="form-check form-check-inline"><input class="perm permn permnproducts" id="permnproducts-1" name="permproducts-1" value="none" label="_" perm_name="n" item_name="products" entity_id="-1" type="radio" disabled="disabled"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permproducts-1"><span class="mdi mdi-close"></span></label></div><div class="form-check form-check-inline"><input class="perm permr permrproducts" id="permrproducts-1" name="permproducts-1" value="none" label="r" perm_name="r" item_name="products" entity_id="-1" type="radio"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permrproducts-1"><span class="mdi mdi-eye mdi-18px"></span></label></div><div class="form-check form-check-inline"><input class="perm permw permwproducts" id="permwproducts-1" name="permproducts-1" value="none" label="rw" perm_name="w" item_name="products" entity_id="-1" type="radio"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permwproducts-1"><span class="mdi mdi-eye mdi-18px"></span><span class="mdi mdi-creation mdi-18px"></span><span class="mdi mdi-border-color mdi-18px"></span><span class="mdi mdi-delete mdi-18px"></span></label></div></div></div><div class="d-flex justify-content-center form-group row" id="permissionsrproducts"><div class="col-sm-6"><div class="iconlabel text-right">restricted products</div></div><div class="col-sm-6"><div class="form-check form-check-inline"><input class="perm permn permnrproducts" id="permnrproducts-1" name="permrproducts-1" value="none" label="_" perm_name="n" item_name="rproducts" entity_id="-1" type="radio"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permrproducts-1"><span class="mdi mdi-close"></span></label></div><div class="form-check form-check-inline"><input class="perm permr permrrproducts" id="permrrproducts-1" name="permrproducts-1" value="none" label="r" perm_name="r" item_name="rproducts" entity_id="-1" type="radio"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permrrproducts-1"><span class="mdi mdi-eye mdi-18px"></span></label></div><div class="form-check form-check-inline"><input class="perm permw permwrproducts" id="permwrproducts-1" name="permrproducts-1" value="none" label="rw" perm_name="w" item_name="rproducts" entity_id="-1" type="radio"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permwrproducts-1"><span class="mdi mdi-eye mdi-18px"></span><span class="mdi mdi-creation mdi-18px"></span><span class="mdi mdi-border-color mdi-18px"></span><span class="mdi mdi-delete mdi-18px"></span></label></div></div></div><div id="permissions"></div><div class="blockquote-footer"><span class="mdi mdi-close">no permission</span><span class="mdi mdi-creation mdi-18px">create</span><span class="mdi mdi-border-color mdi-18px">update</span><span class="mdi mdi-delete mdi-18px">delete</span></div></form><button id="save" class="btn btn-link" type="button" onclick="savePerson()"><span class="mdi mdi-content-save mdi-24px iconlabel">`
	index_10__20 = `</span></button><button class="btn btn-link" type="button" onclick="closeView();"><span class="mdi mdi-close-box mdi-24px iconlabel">`
	index_10__39 = `"></script><script src="../js/chim/person.js"></script></body></html>`
)

func Personindex(c ViewContainer, wr io.Writer) {
	buffer := &WriterAsBuffer{wr}

	buffer.WriteString(index__0)
	WriteAll(c.ProxyPath+"css/bootstrap.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/bootstrap-table.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/select2.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/bootstrap-colorpicker.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/fontawesome.all.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/chimitheque.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/materialdesignicons.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/bootstrap-toggle.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/animate.min.css", true, buffer)
	buffer.WriteString(index__9)
	WriteAll(c.ProxyPath+"js/jquery-3.3.1.min.js", true, buffer)
	buffer.WriteString(index__10)
	WriteAll(c.ProxyPath+"img/logo_chimitheque_small.png", true, buffer)
	buffer.WriteString(index__11)

	if HasPermission(c.PersonID, "r", "products", -2) {
		buffer.WriteString(index__35)
		WriteAll(c.ProxyPath+"v/products", true, buffer)
		buffer.WriteString(index__36)
		WriteAll(T("menu_home", 1), true, buffer)
		buffer.WriteString(index__37)

	}
	if HasPermission(c.PersonID, "r", "products", -2) {
		buffer.WriteString(index__35)
		WriteAll(c.ProxyPath+"v/products?bookmark=true", true, buffer)
		buffer.WriteString(index__39)
		WriteAll(T("menu_bookmark", 1), true, buffer)
		buffer.WriteString(index__37)

	}
	if HasPermission(c.PersonID, "w", "products", -2) {
		buffer.WriteString(index__41)
		WriteAll(c.ProxyPath+"vc/products", true, buffer)
		buffer.WriteString(index__42)
		WriteAll(T("menu_create_productcard", 1), true, buffer)
		buffer.WriteString(index__37)

	}
	if HasPermission(c.PersonID, "r", "entities", -2) {
		buffer.WriteString(index__44)
		WriteAll(T("menu_entity", 1), true, buffer)
		buffer.WriteString(index__45)
		WriteAll(c.ProxyPath+"v/entities", true, buffer)
		buffer.WriteString(index__46)

		if HasPermission(c.PersonID, "all", "all", -1) {
			buffer.WriteString(index__48)
			WriteAll(c.ProxyPath+"vc/entities", true, buffer)
			buffer.WriteString(index__49)

		}
		buffer.WriteString(index__47)

	}
	if HasPermission(c.PersonID, "r", "storages", -2) {
		buffer.WriteString(index__50)
		WriteAll(T("menu_storelocation", 1), true, buffer)
		buffer.WriteString(index__45)
		WriteAll(c.ProxyPath+"v/storelocations", true, buffer)
		buffer.WriteString(index__46)

		if HasPermission(c.PersonID, "all", "all", -2) {
			buffer.WriteString(index__48)
			WriteAll(c.ProxyPath+"vc/storelocations", true, buffer)
			buffer.WriteString(index__49)

		}
		buffer.WriteString(index__47)

	}
	if HasPermission(c.PersonID, "r", "people", -2) {
		buffer.WriteString(index__56)
		WriteAll(T("menu_people", 1), true, buffer)
		buffer.WriteString(index__45)
		WriteAll(c.ProxyPath+"v/people", true, buffer)
		buffer.WriteString(index__46)

		if HasPermission(c.PersonID, "w", "people", -2) {
			buffer.WriteString(index__48)
			WriteAll(c.ProxyPath+"vc/people", true, buffer)
			buffer.WriteString(index__49)

		}
		buffer.WriteString(index__47)

	}
	buffer.WriteString(index__12)
	WriteAll(c.ProxyPath+"vu/peoplepass", true, buffer)
	buffer.WriteString(index__13)
	WriteAll(T("menu_password", 1), true, buffer)
	buffer.WriteString(index__14)
	WriteAll(c.ProxyPath+"delete-token", true, buffer)
	buffer.WriteString(index__15)
	WriteAll(T("menu_logout", 1), true, buffer)
	buffer.WriteString(index_10__16)

	{
		var (
			iconitem   = "border-color"
			iconaction = "account-group"
			label      = "update user"
		)

		buffer.WriteString(index_2__68)
		WriteEscString("mdi-"+iconitem+" mdi mdi-48px", buffer)
		buffer.WriteString(index_2__69)
		WriteEscString("mdi-"+iconaction+" mdi mdi-18px", buffer)
		buffer.WriteString(index_2__70)
		WriteEscString(label, buffer)
		buffer.WriteString(index_2__71)

	}

	buffer.WriteString(index_10__17)

	{
		var (
			label = "email"
			name  = "person_email"
		)

		buffer.WriteString(index_2__72)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__73)
		WriteEscString(label, buffer)
		buffer.WriteString(index_2__74)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__75)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__76)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__77)
	}

	buffer.WriteString(index_4__20)

	{
		var (
			label = "entity(ies)"
			name  = "entities"
		)

		buffer.WriteString(index_2__72)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__73)
		WriteEscString(label, buffer)
		buffer.WriteString(index_2__86)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__75)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__88)

	}

	buffer.WriteString(index_10__19)
	WriteAll(T("save", 1), true, buffer)
	buffer.WriteString(index_10__20)
	WriteAll(T("close", 1), true, buffer)
	buffer.WriteString(index_2__22)

	json, _ := json.Marshal(c)

	var out string
	for key, value := range c.URLValues {
		out += fmt.Sprintf("URLValues.set(%s, %s)\n", key, value)
	}

	buffer.WriteString(index__17)
	WriteAll(c.ProxyPath, false, buffer)
	buffer.WriteString(index__18)
	buffer.WriteString(fmt.Sprintf("%s", json))
	buffer.WriteString(index__19)
	buffer.WriteString(out)
	buffer.WriteString(index__20)
	WriteAll(c.ProxyPath+"js/jquery.formautofill.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/jquery.validate.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/jquery.validate.additional-methods.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/select2.full.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/popper.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/bootstrap.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/bootstrap-table.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/bootstrap-confirmation.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/bootstrap-colorpicker.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/bootstrap-toggle.min.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/JSmol.lite.nojq.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/chim/gjs-common.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/chim/chimcommon.js", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(c.ProxyPath+"js/chim/login.js", true, buffer)
	buffer.WriteString(index_10__39)

}
