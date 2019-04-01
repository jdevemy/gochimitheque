// Code generated by "jade.go"; DO NOT EDIT.

package jade

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	index_1__9  = `"></script></head><body><div id="message"></div><div class="container"><footer class="row"><div class="col-sm-12"><span class="text-right blockquote-footer" id="logged"></span></div></footer><header class="row justify-content-left"><div class="col-sm-12"><img src="`
	index_1__10 = `" alt="chimitheque_logo" title="Chimithèque"/></div></header><header><form id="authForm"><div class="row"><div class="col col-sm-4 offset-sm-4"><div class="form-group"><label for="person_email"></label>`
	index_1__11 = `<input class="form-control" id="person_email" type="email" aria-describedby="emailHelp" placeholder="`
	index_1__12 = `" name="person_email"/></div><div class="form-group"><label for="person_password"></label>`
	index_1__13 = `<input class="form-control" id="person_password" type="password" aria-describedby="passwordHelp" placeholder="`
	index_1__14 = `" name="person_password"/></div></div></div><div class="row"><div class="col offset-sm-4 col-sm-2"><a id="gettoken" href="#" onclick="getToken();"><span class="mdi mdi-36px mdi-login iconlabel">`
	index_1__15 = `</span></a></div><div class="col col-sm-2"><p class="text-right"><a id="getcaptcha" href="#" onclick="getCaptcha();"><span class="mdi mdi-36px mdi-lock-reset iconlabel">`
	index_1__16 = `</span></a></p></div></div></form><form id="captcha"><div class="row invisible" id="captcha-row"><div class="col col-sm-12 d-flex justify-content-center mt-sm-4 mb-sm-2"><img id="captcha-img"/></div><div class="col col-sm-12 d-flex justify-content-center mb-sm-4"><input id="captcha_uid" type="hidden" name="captcha_uid"/><input id="captcha_text" type="text" name="captcha_text"/><a id="resetpassword" href="#" onclick="resetPassword();"><span class="mdi mdi-36px mdi-lock-reset iconlabel">`
	index_1__17 = `</span></a></div></div></form></header><a href="https://github.com/tbellembois/gochimitheque"><img style="position: absolute; top: 0; right: 0; border: 0;" src="`
	index_1__18 = `" alt="Fork me on GitHub"/></a><div class="fixed-bottom row"><div class="col col-sm-3"><img class="mx-auto d-block" src="`
	index_1__19 = `" alt="ens_logo" title="Ecole Normale Supérieure de Lyon"/></div><div class="col col-sm-3"><img class="mx-auto d-block" src="`
	index_1__20 = `" alt="uca_logo" title="Université Clermont Auvergne"/></div><div class="col col-sm-3"><img class="mx-auto d-block" src="`
	index_1__21 = `" alt="go_logo" title="Golang language"/></div><div class="col col-sm-3"><i>`
	index_1__22 = `</i><a title="mon-aloevera [at] hotmail [dot] com"><b>Katia Varet.</b></a><p class="blockquote-footer"><i>`
	index_1__23 = `</i></p></div></div></div><!--  Code generated by go generate; DO NOT EDIT. --><script>    
	var locale_en_advancedsearch_text = "advanced search";
	
	var locale_en_casnumber_cmr_title = "CMR";
	
	var locale_en_casnumber_label_title = "CAS";
	
	var locale_en_cenumber_label_title = "CE";
	
	var locale_en_classofcompound_label_title = "class of compounds";
	
	var locale_en_clearsearch_text = "clear search form";
	
	var locale_en_close = "close";
	
	var locale_en_create = "create";
	
	var locale_en_created = "created";
	
	var locale_en_createperson_mailsubject = "Chimithèque new account\r\n";
	
	var locale_en_delete = "delete";
	
	var locale_en_edit = "edit";
	
	var locale_en_email_placeholder = "enter your email";
	
	var locale_en_empiricalformula_label_title = "empirical formula";
	
	var locale_en_entity_create_title = "create entity";
	
	var locale_en_entity_created_message = "entity created";
	
	var locale_en_entity_deleted_message = "entity deleted";
	
	var locale_en_entity_description_table_header = "description";
	
	var locale_en_entity_manager_table_header = "manager(s)";
	
	var locale_en_entity_name_table_header = "name";
	
	var locale_en_entity_nameexist_validate = "entity with this name already present";
	
	var locale_en_entity_updated_message = "entity updated";
	
	var locale_en_export_text = "export";
	
	var locale_en_hazardstatement_label_title = "hazard statement(s)";
	
	var locale_en_hidedeleted_text = "hide deleted";
	
	var locale_en_linearformula_label_title = "liner formula";
	
	var locale_en_list = "list";
	
	var locale_en_logo_information1 = "Chimithèque logo designed by ";
	
	var locale_en_logo_information2 = "Do not use or copy without her permission.";
	
	var locale_en_members = "members";
	
	var locale_en_menu_bookmark = "my bookmarks";
	
	var locale_en_menu_create_productcard = "create product card";
	
	var locale_en_menu_entity = "entities";
	
	var locale_en_menu_home = "home";
	
	var locale_en_menu_logout = "logout";
	
	var locale_en_menu_password = "change my password";
	
	var locale_en_menu_people = "people";
	
	var locale_en_menu_storelocation = "store locations";
	
	var locale_en_modified = "modified";
	
	var locale_en_password_placeholder = "enter your password";
	
	var locale_en_physicalstate_label_title = "physical state";
	
	var locale_en_precautionarystatement_label_title = "precautionary statement(s)";
	
	var locale_en_product_disposalcomment_title = "disposal comment";
	
	var locale_en_product_msds_title = "MSDS";
	
	var locale_en_product_radioactive_title = "radioactive";
	
	var locale_en_product_remark_title = "remark";
	
	var locale_en_product_restricted_title = "restricted access";
	
	var locale_en_product_threedformula_title = "3D formula";
	
	var locale_en_required_input = "required input";
	
	var locale_en_resetpassword2_text = "reset my password, I am not a robot";
	
	var locale_en_resetpassword_areyourobot = "are you a robot?";
	
	var locale_en_resetpassword_done = "A new temporary password has been sent to %s";
	
	var locale_en_resetpassword_mailsubject1 = "Chimithèque new temporary password\r\n";
	
	var locale_en_resetpassword_mailsubject2 = "Chimithèque password reset link\r\n";
	
	var locale_en_resetpassword_message_mailsentto = "a reinitialization link has been sent to";
	
	var locale_en_resetpassword_text = "reset password";
	
	var locale_en_resetpassword_warning_enteremail = "enter your email in the login form";
	
	var locale_en_s_casnumber = "CAS";
	
	var locale_en_s_casnumber_cmr = "CMR";
	
	var locale_en_s_custom_name_part_of = "part of name";
	
	var locale_en_s_empiricalformula = "emp. formula";
	
	var locale_en_s_hazardstatements = "hazard statement(s)";
	
	var locale_en_s_name = "name";
	
	var locale_en_s_precautionarystatements = "precautionary statement(s)";
	
	var locale_en_s_signalword = "signal word";
	
	var locale_en_s_storage_barecode = "barecode";
	
	var locale_en_s_symbols = "symbol(s)";
	
	var locale_en_save = "save";
	
	var locale_en_search_text = "search";
	
	var locale_en_showdeleted_text = "show deleted";
	
	var locale_en_signalword_label_title = "signal word";
	
	var locale_en_stock_storelocation_sub_title = "including children store locations";
	
	var locale_en_stock_storelocation_title = "in this store location";
	
	var locale_en_storage_barecode_title = "barecode";
	
	var locale_en_storage_batchnumber_title = "batch number";
	
	var locale_en_storage_borrow = "borrow";
	
	var locale_en_storage_clone = "clone";
	
	var locale_en_storage_comment_title = "comment";
	
	var locale_en_storage_entrydate_title = "entry date";
	
	var locale_en_storage_exitdate_title = "exit date";
	
	var locale_en_storage_expirationdate_title = "expiration date";
	
	var locale_en_storage_openingdate_title = "opening date";
	
	var locale_en_storage_quantity_title = "quantity";
	
	var locale_en_storage_restore = "restore";
	
	var locale_en_storage_showhistory = "show history";
	
	var locale_en_storage_unborrow = "unborrow";
	
	var locale_en_storelocations = "storelocations";
	
	var locale_en_submitlogin_text = "enter";
	
	var locale_en_supplier_label_title = "supplier";
	
	var locale_en_switchproductview_text = "switch to product view";
	
	var locale_en_switchstorageview_text = "switch to storage view";
	
	var locale_en_test = "One test";
	
    
	var locale_fr_advancedsearch_text = "recherche avancée";
	
	var locale_fr_casnumber_cmr_title = "CMR";
	
	var locale_fr_casnumber_label_title = "CAS";
	
	var locale_fr_cenumber_label_title = "CE";
	
	var locale_fr_classofcompound_label_title = "famille chimique";
	
	var locale_fr_clearsearch_text = "effacer le formulaire";
	
	var locale_fr_close = "fermer";
	
	var locale_fr_create = "créer";
	
	var locale_fr_created = "créé";
	
	var locale_fr_createperson_mailsubject = "Chimithèque nouveau compte\r\n";
	
	var locale_fr_delete = "supprimer";
	
	var locale_fr_edit = "editer";
	
	var locale_fr_email_placeholder = "entrez votre email";
	
	var locale_fr_empiricalformula_label_title = "formule brute";
	
	var locale_fr_entity_create_title = "créer une entité";
	
	var locale_fr_entity_created_message = "entité crée";
	
	var locale_fr_entity_deleted_message = "entité supprimée";
	
	var locale_fr_entity_description_table_header = "description";
	
	var locale_fr_entity_manager_table_header = "responsable(s)";
	
	var locale_fr_entity_name_table_header = "nom";
	
	var locale_fr_entity_nameexist_validate = "une entité avec ce nom existe déjà";
	
	var locale_fr_entity_updated_message = "entité mise à jour";
	
	var locale_fr_export_text = "exporter";
	
	var locale_fr_hazardstatement_label_title = "mention(s) de danger H-EUH";
	
	var locale_fr_hidedeleted_text = "cacher supprimés";
	
	var locale_fr_linearformula_label_title = "formule linéaire";
	
	var locale_fr_list = "lister";
	
	var locale_fr_logo_information1 = "Logo Chimithèque réalisé par ";
	
	var locale_fr_logo_information2 = "Ne pas utiliser ou copier sans sa permission.";
	
	var locale_fr_members = "membres";
	
	var locale_fr_menu_bookmark = "mes favoris";
	
	var locale_fr_menu_create_productcard = "créer fiche produit";
	
	var locale_fr_menu_entity = "entités";
	
	var locale_fr_menu_home = "accueil";
	
	var locale_fr_menu_logout = "déconnexion";
	
	var locale_fr_menu_password = "changer mon mot de passe";
	
	var locale_fr_menu_people = "utilisateurs";
	
	var locale_fr_menu_storelocation = "entrepôts";
	
	var locale_fr_modified = "modifié";
	
	var locale_fr_password_placeholder = "entrez votre mot de passe";
	
	var locale_fr_physicalstate_label_title = "état physique";
	
	var locale_fr_precautionarystatement_label_title = "conseil(s) de prudence P";
	
	var locale_fr_product_disposalcomment_title = "commentaire de destruction";
	
	var locale_fr_product_msds_title = "FDS";
	
	var locale_fr_product_radioactive_title = "radioactif";
	
	var locale_fr_product_remark_title = "remarque";
	
	var locale_fr_product_restricted_title = "accès restreint";
	
	var locale_fr_product_threedformula_title = "formule 3D";
	
	var locale_fr_required_input = "champs requis";
	
	var locale_fr_resetpassword2_text = "réinitialiser mon mot de passe, je ne suis pas un robot";
	
	var locale_fr_resetpassword_areyourobot = "êtes vous un robot ?";
	
	var locale_fr_resetpassword_done = "Un nouveau mot de passe temporaire a été envoyé à %s";
	
	var locale_fr_resetpassword_mailsubject1 = "Chimithèque nouveau mot de passe temporaire\r\n";
	
	var locale_fr_resetpassword_mailsubject2 = "Chimithèque lien de réinitialisation de mot de passe\r\n";
	
	var locale_fr_resetpassword_message_mailsentto = "un mail de réinitialisation a été envoyé à";
	
	var locale_fr_resetpassword_text = "réinitialiser mon mot de passe";
	
	var locale_fr_resetpassword_warning_enteremail = "entrez votre adresse mail dans le formulaire";
	
	var locale_fr_s_casnumber = "CAS";
	
	var locale_fr_s_custom_name_part_of = "partie du nom";
	
	var locale_fr_s_empiricalformula = "formule brute";
	
	var locale_fr_s_hazardstatements = "mention(s) de danger H-EUH";
	
	var locale_fr_s_name = "nom";
	
	var locale_fr_s_precautionarystatements = "conseil(s) de prudence P";
	
	var locale_fr_s_signalword = "mention d'avertissement";
	
	var locale_fr_s_storage_barecode = "code barre";
	
	var locale_fr_s_symbols = "symbole(s)";
	
	var locale_fr_save = "enregistrer";
	
	var locale_fr_search_text = "rechercher";
	
	var locale_fr_showdeleted_text = "voir supprimés";
	
	var locale_fr_signalword_label_title = "mention d'avertissement";
	
	var locale_fr_storage_barecode_title = "code barre";
	
	var locale_fr_storage_batchnumber_title = "numéro de lot";
	
	var locale_fr_storage_borrow = "emprunter";
	
	var locale_fr_storage_clone = "cloner";
	
	var locale_fr_storage_comment_title = "commentaire";
	
	var locale_fr_storage_entrydate_title = "date d'entrée";
	
	var locale_fr_storage_exitdate_title = "date de sortie";
	
	var locale_fr_storage_expirationdate_title = "date d'expiration";
	
	var locale_fr_storage_openingdate_title = "date d'ouverture";
	
	var locale_fr_storage_quantity_title = "quantité";
	
	var locale_fr_storage_restore = "restaurer";
	
	var locale_fr_storage_showhistory = "voir historique";
	
	var locale_fr_storage_unborrow = "restituer";
	
	var locale_fr_storelocations = "entrepôts";
	
	var locale_fr_submitlogin_text = "entrer";
	
	var locale_fr_supplier_label_title = "fournisseur";
	
	var locale_fr_switchproductview_text = "vue par produits";
	
	var locale_fr_switchstorageview_text = "vue par stockages";
	
	var locale_fr_test = "Un test";
	</script>`
	index_1__25 = `";

            // logged user email and permissions
            var container = `
	index_1__27 = `

            // setting up logged user
            window.onload = function() {
                //var email = readCookie("email")
                var urlParams = new URLSearchParams(window.location.search);
                var message = urlParams.get("message");
                
                //- if (email != null) {
                //-     document.getElementById("logged").innerHTML = email;
                //- }
                if (message != null) {
                    global.displayMessage(message, "success");
                }

           };
</script><script src="`
	index_1__39 = `"></script><script src="../js/chim/login.js"></script></body></html>`
)

func Login(c ViewContainer, wr io.Writer) {
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
	WriteAll(c.ProxyPath+"css/chimitheque.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/materialdesignicons.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/bootstrap-toggle.min.css", true, buffer)
	buffer.WriteString(index__1)
	WriteAll(c.ProxyPath+"css/animate.min.css", true, buffer)
	buffer.WriteString(index__9)
	WriteAll(c.ProxyPath+"js/jquery-3.3.1.min.js", true, buffer)
	buffer.WriteString(index_1__9)
	WriteAll(c.ProxyPath+"img/logo_chimitheque.png", true, buffer)
	buffer.WriteString(index_1__10)

	var a, b = "email_placeholder", 1
	buffer.WriteString(index_1__11)
	WriteAll(T(a, b), true, buffer)
	buffer.WriteString(index_1__12)

	a, b = "password_placeholder", 1
	buffer.WriteString(index_1__13)
	WriteAll(T(a, b), true, buffer)
	buffer.WriteString(index_1__14)
	WriteAll(T("submitlogin_text", 1), true, buffer)
	buffer.WriteString(index_1__15)
	WriteAll(T("resetpassword_text", 1), true, buffer)
	buffer.WriteString(index_1__16)
	WriteAll(T("resetpassword2_text", 1), true, buffer)
	buffer.WriteString(index_1__17)
	WriteAll(c.ProxyPath+"img/forkme_right_darkblue_121621.png", true, buffer)
	buffer.WriteString(index_1__18)
	WriteAll(c.ProxyPath+"img/logo_ens.png", true, buffer)
	buffer.WriteString(index_1__19)
	WriteAll(c.ProxyPath+"img/logo_uca.jpg", true, buffer)
	buffer.WriteString(index_1__20)
	WriteAll(c.ProxyPath+"img/logo_go.png", true, buffer)
	buffer.WriteString(index_1__21)
	WriteAll(T("logo_information1", 1), true, buffer)
	buffer.WriteString(index_1__22)
	WriteAll(T("logo_information2", 1), true, buffer)
	buffer.WriteString(index_1__23)

	json, _ := json.Marshal(c)

	var out string
	for key, value := range c.URLValues {
		out += fmt.Sprintf("URLValues.set(%s, %s)\n", key, value)
	}

	buffer.WriteString(index__37)
	WriteAll(c.ProxyPath, false, buffer)
	buffer.WriteString(index_1__25)
	buffer.WriteString(fmt.Sprintf("%s", json))
	buffer.WriteString(index__39)
	buffer.WriteString(out)
	buffer.WriteString(index_1__27)
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
	WriteAll(c.ProxyPath+"js/chim/gjs-common.js", true, buffer)
	buffer.WriteString(index__41)
	WriteAll(c.ProxyPath+"js/chim/chimcommon.js", true, buffer)
	buffer.WriteString(index_1__39)

}
