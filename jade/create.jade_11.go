// Code generated by "jade.go"; DO NOT EDIT.

package jade

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	create_11__37 = `<form id="person"><div class="row"><div class="col-sm-6">`
	create_11__39 = `</div></div><span>permissions</span><div class="d-flex justify-content-center form-group row" id="permissionsproducts"><div class="col-sm-6"><div class="iconlabel text-right">products</div></div><div class="col-sm-6"><div class="form-check form-check-inline"><input class="perm permn permnproducts" id="permnproducts-1" name="permproducts-1" value="none" label="_" perm_name="n" item_name="products" entity_id="-1" type="radio" disabled="disabled"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permproducts-1"><span class="mdi mdi-close"></span></label></div><div class="form-check form-check-inline"><input class="perm permr permrproducts" id="permrproducts-1" name="permproducts-1" value="none" label="r" perm_name="r" item_name="products" entity_id="-1" type="radio" checked="checked"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permrproducts-1"><span class="mdi mdi-eye mdi-18px"></span></label></div><div class="form-check form-check-inline"><input class="perm permw permwproducts" id="permwproducts-1" name="permproducts-1" value="none" label="rw" perm_name="w" item_name="products" entity_id="-1" type="radio"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permwproducts-1"><span class="mdi mdi-eye mdi-18px"></span><span class="mdi mdi-creation mdi-18px"></span><span class="mdi mdi-border-color mdi-18px"></span><span class="mdi mdi-delete mdi-18px"></span></label></div></div></div><div class="d-flex justify-content-center form-group row" id="permissionsrproducts"><div class="col-sm-6"><div class="iconlabel text-right">restricted products</div></div><div class="col-sm-6"><div class="form-check form-check-inline"><input class="perm permn permnrproducts" id="permnrproducts-1" name="permrproducts-1" value="none" label="_" perm_name="n" item_name="rproducts" entity_id="-1" type="radio" checked="checked"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permrproducts-1"><span class="mdi mdi-close"></span></label></div><div class="form-check form-check-inline"><input class="perm permr permrrproducts" id="permrrproducts-1" name="permrproducts-1" value="none" label="r" perm_name="r" item_name="rproducts" entity_id="-1" type="radio"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permrrproducts-1"><span class="mdi mdi-eye mdi-18px"></span></label></div><div class="form-check form-check-inline"><input class="perm permw permwrproducts" id="permwrproducts-1" name="permrproducts-1" value="none" label="rw" perm_name="w" item_name="rproducts" entity_id="-1" type="radio"/><label class="form-check-label ml-sm-1 pr-sm-1 pl-sm-1 text-secondary border border-secondary rounded" for="permwrproducts-1"><span class="mdi mdi-eye mdi-18px"></span><span class="mdi mdi-creation mdi-18px"></span><span class="mdi mdi-border-color mdi-18px"></span><span class="mdi mdi-delete mdi-18px"></span></label></div></div></div><div id="permissions"></div><div class="blockquote-footer"><span class="mdi mdi-close">no permission</span><span class="mdi mdi-creation mdi-18px">create</span><span class="mdi mdi-border-color mdi-18px">update</span><span class="mdi mdi-delete mdi-18px">delete</span></div></form><button class="btn btn-link" type="button" onclick="savePerson()"><span class="mdi mdi-content-save mdi-24px iconlabel">`
)

func Personcreate(c ViewContainer, wr io.Writer) {
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
	WriteAll(c.ProxyPath+"v/products", true, buffer)
	buffer.WriteString(index__12)
	WriteAll(T("menu_home", 1), true, buffer)
	buffer.WriteString(index__13)
	WriteAll(c.ProxyPath+"v/products?bookmark=true", true, buffer)
	buffer.WriteString(index__14)
	WriteAll(T("menu_bookmark", 1), true, buffer)
	buffer.WriteString(index__15)
	WriteAll(c.ProxyPath+"vc/products", true, buffer)
	buffer.WriteString(index__16)
	WriteAll(T("menu_create_productcard", 1), true, buffer)
	buffer.WriteString(index__17)
	WriteAll(T("menu_entity", 1), true, buffer)
	buffer.WriteString(index__18)
	WriteAll(c.ProxyPath+"v/entities", true, buffer)
	buffer.WriteString(index__19)
	WriteAll(T("list", 1), true, buffer)
	buffer.WriteString(index__20)
	WriteAll(c.ProxyPath+"vc/entities", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(T("create", 1), true, buffer)
	buffer.WriteString(index__22)
	WriteAll(T("menu_storelocation", 1), true, buffer)
	buffer.WriteString(index__18)
	WriteAll(c.ProxyPath+"v/storelocations", true, buffer)
	buffer.WriteString(index__19)
	WriteAll(T("list", 1), true, buffer)
	buffer.WriteString(index__25)
	WriteAll(c.ProxyPath+"vc/storelocations", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(T("create", 1), true, buffer)
	buffer.WriteString(index__27)
	WriteAll(T("menu_people", 1), true, buffer)
	buffer.WriteString(index__18)
	WriteAll(c.ProxyPath+"v/people", true, buffer)
	buffer.WriteString(index__19)
	WriteAll(T("list", 1), true, buffer)
	buffer.WriteString(index__30)
	WriteAll(c.ProxyPath+"vc/people", true, buffer)
	buffer.WriteString(index__21)
	WriteAll(T("create", 1), true, buffer)
	buffer.WriteString(index__32)
	WriteAll(c.ProxyPath+"vu/peoplepass", true, buffer)
	buffer.WriteString(index__33)
	WriteAll(T("menu_password", 1), true, buffer)
	buffer.WriteString(index__34)
	WriteAll(c.ProxyPath+"delete-token", true, buffer)
	buffer.WriteString(index__35)
	WriteAll(T("menu_logout", 1), true, buffer)
	buffer.WriteString(create__36)

	{
		var (
			iconitem   = "creation"
			iconaction = "account-group"
			label      = "create user"
		)

		buffer.WriteString(index_2__64)
		WriteEscString("mdi-"+iconitem+" mdi mdi-48px", buffer)
		buffer.WriteString(index_2__65)
		WriteEscString("mdi-"+iconaction+" mdi mdi-18px", buffer)
		buffer.WriteString(index_2__66)
		WriteEscString(label, buffer)
		buffer.WriteString(index_2__67)

	}

	buffer.WriteString(create_11__37)

	{
		var (
			label = "email"
			name  = "person_email"
		)

		buffer.WriteString(index_2__68)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__69)
		WriteEscString(label, buffer)
		buffer.WriteString(index_2__70)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__71)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__72)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__73)
	}

	buffer.WriteString(index_4__40)

	{
		var (
			label = "entity(ies)"
			name  = "entities"
		)

		buffer.WriteString(index_2__68)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__69)
		WriteEscString(label, buffer)
		buffer.WriteString(index_2__82)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__71)
		WriteEscString(name, buffer)
		buffer.WriteString(index_2__84)

	}

	buffer.WriteString(create_11__39)
	WriteAll(T("save", 1), true, buffer)
	buffer.WriteString(create__41)

	json, _ := json.Marshal(c)

	var out string
	for key, value := range c.URLValues {
		out += fmt.Sprintf("URLValues.set(%s, %s)\n", key, value)
	}

	buffer.WriteString(index__37)
	WriteAll(c.ProxyPath, false, buffer)
	buffer.WriteString(index__38)
	buffer.WriteString(fmt.Sprintf("%s", json))
	buffer.WriteString(index__39)
	buffer.WriteString(out)
	buffer.WriteString(index__40)
	WriteAll(c.ProxyPath+"js/jquery.formautofill.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/jquery.validate.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/jquery.validate.additional-methods.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/select2.full.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/popper.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/bootstrap.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/bootstrap-table.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/bootstrap-confirmation.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/bootstrap-colorpicker.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/bootstrap-toggle.min.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/JSmol.lite.nojq.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/chim/gjs-common.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/chim/chimcommon.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/chim/login.js", true, buffer)
	buffer.WriteString(index_10__59)

}
