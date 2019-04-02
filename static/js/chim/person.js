$( document ).ready(function() {

    //
    // INITIAL SETUP
    //

    // populating search input if needed
    if (URLValues.has("search")) {
        $('#table').bootstrapTable('resetSearch', URLValues.get("search")[0]);
    }

    //
    // FORM VALIDATION
    //

    $( "#person" ).validate({
        errorClass: "alert alert-danger",
        rules: {
            person_email: {
                required: true,
                email: true,
                remote: {
                    url: "",
                    type: "post",
                    beforeSend: function(jqXhr, settings) {
                        id = -1
                        if ($("form#person input#person_id").length) {
                            id = $("form#person input#person_id").val()
                        }
                        settings.url = proxyPath + "validate/person/" + id + "/email/";
                    },
                },
            },
        },
    });
    $( "#personp" ).validate({
        errorClass: "alert alert-danger",
        rules: {
            person_password: {
                required: true,
            },
            person_passwordagain: {
                equalTo: "#person_password",
            },
        },
        messages: {
            person_password: {
                required: "enter your new password",
            },
            person_passwordagain: {
                equalTo: "you have not entered the same password",
            },
        },
    });

    //
    // SELECT2 SETUP
    //
    
    // entities select2
    $('select#entities').select2({
        ajax: {
            url: proxyPath + 'entities',
            dataType: 'json',
            data: function (params) {
                var query = {
                    search: params.term,
                    page: params.page || 1,
                    offset: (params.page-1)*10 || 0,
                    limit: 10
                }

                // Query parameters will be ?search=[term]&page=[page]
                return query;
            },
            processResults: function (data) {
                // replacing email by text expected by select2
                var newdata = $.map(data.rows, function (obj) {
                    obj.text = obj.text || obj.entity_name;
                    obj.id = obj.id || obj.entity_id;
                    return obj;
                });
                // getting the number of loaded select elements
                selectnbitems = $("ul#select2-entities-results li").length + 10;

                return {
                    results: newdata,
                    pagination: {more: selectnbitems<data.total}
                };
            }
        }
    });
    $('select#entities').on('select2:unselecting', function (e) {
        var ismanager = false;
        var data = e.params.args.data,
            entityid = data.id,
            entityname = data.text;
        
        // preventing unselecting entity if the person is one its manager
        manageentities = $("input.manageentities")
        $.each(manageentities, function( index, e ) {
            if ($(e).val() == entityid) {
                ismanager = true;
            }
        });
        if (ismanager) {
            global.displayMessage("this entity can not be removed, the user is one of its manager", "success");
            e.preventDefault();
        } else {
            // removing permissions widget
            $("#perm" + data.id).remove();
        }
    });
    $('select#entities').on('select2:select', function (e) {
        var data = e.params.data;
        
        // adding permissions widget
        $("#permissions").append(global.buildPermissionWidget(data.entity_id, data.entity_name));
    });
    
});

//
// TABLE SETUP
//

// custom table row attribute
function rowAttributes(row, index) {
    return {"person_id":row["person_id"]}
}

// table data loading
function getData(params) {
    $.ajax({
        url: proxyPath + "people",
        method: "GET",
        dataType: "JSON",
        data: params.data,
    }).done(function(data, textStatus, jqXHR) {
        params.success({
            rows: data.rows,
            total: data.total,
        });
    }).fail(function(jqXHR, textStatus, errorThrown) {
        params.error(jqXHR.statusText);                
        handleHTTPError(jqXHR.statusText, jqXHR.status)
    });           
}
function queryParams(params) {
    // getting request parameters
    var urlParams = new URLSearchParams(window.location.search);

    if (urlParams.has("entity")) {
        params["entity"] = urlParams.get("entity");
    }
    return params;
}

// when table is loaded
$('#table').on('load-success.bs.table refresh.bs.table', function () {
    $("table#table").find("tr").each(function( index, b ) {
        hasPermission("people", $(b).attr("person_id"), "GET").done(function(){
            $("#view"+$(b).attr("person_id")).fadeIn();
            localStorage.setItem("people:" + $(b).attr("person_id") + ":GET", true);
        }).fail(function(){
            localStorage.setItem("people:" + $(b).attr("person_id") + ":GET", false);
        })
    });
    $("table#table").find("tr").each(function( index, b ) {
        hasPermission("people", $(b).attr("person_id"), "PUT").done(function(){
            $("#edit"+$(b).attr("person_id")).fadeIn();
            localStorage.setItem("people:" + $(b).attr("person_id") + ":PUT", true);
        }).fail(function(){
            localStorage.setItem("people:" + $(b).attr("person_id") + ":PUT", false);
        })
    });
    $("table#table").find("tr").each(function( index, b ) {
        hasPermission("people", $(b).attr("person_id"), "DELETE").done(function(){
            $("#delete"+$(b).attr("person_id")).fadeIn();
            localStorage.setItem("people:" + $(b).attr("person_id") + ":DELETE", true);
        }).fail(function(){
            localStorage.setItem("people:" + $(b).attr("person_id") + ":DELETE", false);
        })
    });
});

//
// TABLE FORMATTERS
//

// actions formatter
function operateFormatter(value, row, index) {
    // show action buttons if permitted
    eid = row.person_id

    // buttons are hidden by default
    var actions = [
    // '<button id="view' + eid + '" eid="' + eid + '" class="view btn btn-link btn-sm" style="display: none;" title="view" type="button">',
    //     '<span class="mdi mdi-eye mdi-24px"></span>',
    // '</button>',
    '<button id="edit' + eid + '" eid="' + eid + '" class="edit btn btn-link btn-sm" style="display: none;" title="edit" type="button">',
        '<span class="mdi mdi-border-color mdi-24px"></span>',
    '</button>',
    '<button id="delete' + eid + '" eid="' + eid + '" class="delete btn btn-link btn-sm" style="display: none;" title="delete" type="button">',
        '<span class="mdi mdi-delete mdi-24px"></span>',
    '</button>'];

    return actions.join('&nbsp;')    
}

//
// TABLE ACTIONS DEFINITION
//
window.operateEvents = {
    // 'click .view': function (e, value, row, index) {
    //     operateEditView(e, value, row, index)
    // },
    'click .edit': function (e, value, row, index) {
        operateEditView(e, value, row, index)
    },
    'click .delete': function (e, value, row, index) {
        // hidding possible previous confirmation button
        $(this).confirmation("show").off( "confirmed.bs.confirmation");
        $(this).confirmation("show").off( "canceled.bs.confirmation");
        
        // ask for confirmation and then delete
        $(this).confirmation("show").on( "confirmed.bs.confirmation", function() {
            $.ajax({
                url: proxyPath + "people/" + row['person_id'],
                method: "DELETE",
            }).done(function(data, textStatus, jqXHR) {
                global.displayMessage("person deleted", "success");
                var $table = $('#table');
                $table.bootstrapTable('refresh');
            }).fail(function(jqXHR, textStatus, errorThrown) {
                handleHTTPError(jqXHR.statusText, jqXHR.status)
            });
        }).on( "canceled.bs.confirmation", function() {
        });
    }
};
function operateEditView(e, value, row, index) {
    // clearing selections
    $('select#entities').val(null).trigger('change');
    $('select#entities').find('option').remove();

    var persondata,
        managedentitydata,
        entitydata,
        permissiondata;
    var managedentitydataids;

    // getting the person
    personpromise = $.ajax({
        url: proxyPath + "people/" + row['person_id'],
        method: "GET",
    }).done(function(data, textStatus, jqXHR) {
        //console.log("done personpromise");
        persondata = data;
    }).fail(function(jqXHR, textStatus, errorThrown) {
        handleHTTPError(jqXHR.statusText, jqXHR.status)
    });

    // getting the entities the person is manager of
    managerpromise = $.ajax({
        url: proxyPath + "people/" + row['person_id'] + "/manageentities",
        method: "GET",
    }).done(function(data, textStatus, jqXHR) {
        //console.log("done managerpromise");
        managedentitydata = data;
        if (data != null) {
            managedentitydataids = data.map(function(a) {return a.entity_id;});
        } else {
            managedentitydataids = [];
        }
    });

    // getting the person permissions
    permissionpromise = $.ajax({
        url: proxyPath + "people/" + row['person_id'] + "/permissions",
        method: "GET",
    }).done(function(data, textStatus, jqXHR) {
        //console.log("done permissionpromise");
        permissiondata = data;
    }).fail(function(jqXHR, textStatus, errorThrown) {
        handleHTTPError(jqXHR.statusText, jqXHR.status)
    });

    // getting the person entities
    entitypromise = $.ajax({
        url: proxyPath + "people/" + row['person_id'] + "/entities",
        method: "GET",
    }).done(function(data, textStatus, jqXHR) {
        //console.log("done entitypromise");
        entitydata = data;
    }).fail(function(jqXHR, textStatus, errorThrown) {
        handleHTTPError(jqXHR.statusText, jqXHR.status)
    });

    $.when(personpromise, managerpromise, entitypromise, permissionpromise).done(function() {
        // flattening person response data
        fdata = flatten(persondata);
        // autofilling form
        $("#viewedit-collapse").autofill( fdata, {"findbyname": false } );
        // setting index hidden input
        $("input#index").val(index);

        // appending managed entities in hidden inputs for further use
        $("input.manageentities").remove();
        for(var i in managedentitydata) {
           var newOption = $("<input></input>");
           newOption.addClass("manageentities");
           newOption.attr("type", "hidden");
           newOption.val(managedentitydata[i].entity_id);
           $('form#person-update').append(newOption);
        }

        // populating the entities select2
        for(var i in entitydata) {
           var newOption = new Option(entitydata[i].entity_name, entitydata[i].entity_id, true, true);
           $('select#entities').append(newOption).trigger('change');
        }
        // adding a permission widget for each entity
        // except for managed entities
        $("#permissions").empty();
        for(var i in entitydata) {
            if ($.inArray(entitydata[i].entity_id, managedentitydataids) == -1){
                $("#permissions").append(global.buildPermissionWidget(entitydata[i].entity_id, entitydata[i].entity_name, false));
            } else {
                $("#permissions").append(global.buildPermissionWidget(entitydata[i].entity_id, entitydata[i].entity_name, true));
            }
        }

        // populating the permissions widget
        if ($("input.perm").length > 0) {
            if (permissiondata == null) {
                permissiondata = [];
            }
            global.populatePermissionWidget(permissiondata);
        }

        // disabling or enabling the widgets
        // if ($(e.target).hasClass("view")) {
        //     $("#viewedit-collapse").find("input").prop("disabled", true);
        //     $("#viewedit-collapse").find("select").prop("disabled", true);
        //     $("#viewedit-collapse").find("button#save").hide();
        // } else {
        //     $("#viewedit-collapse").find("input").prop("disabled", false);
        //     $("#viewedit-collapse").find("select").prop("disabled", false);
        //     $("#viewedit-collapse").find("button#save").show();
        // }

        // finally collapsing the view
        $('#viewedit-collapse').collapse('show');
        $('#list-collapse').collapse('hide');
    });
};

//
// close buttons actions
//
function closeView() { $("#list-collapse").collapse("show"); $("#viewedit-collapse").collapse("hide"); }

//
// save person callback
//
var createCallBack = function createCallback(data, textStatus, jqXHR) {
    global.displayMessage("person " + data.person_email + " created", "success");
    setTimeout(function(){ window.location = proxyPath + "v/people"; }, 1000);
}
var updateCallBack = function updateCallback(data, textStatus, jqXHR) {
    $('#list-collapse').collapse('show');
    $('#viewedit-collapse').collapse('hide');
    var $table = $('#table');
    var index = $('input#index').val();
    $table.bootstrapTable('updateRow', {
        index: index,
        row: {
            "person_email": data.person_email,
        }
    });
    global.displayMessage("person " + data.person_email + " updated", "success");
}
function savePersonp() {
    var form = $("#personp");
    if (! form.valid()) {
        return;
    };

    var person_password = $("input#person_password").val();

    $.ajax({
        url: proxyPath + "peoplep",
        method: "POST",
        dataType: 'json',
        data: {"person_password": person_password},
    }).done(function(jqXHR, textStatus, errorThrown) {
        global.displayMessage("password updated", "success");
        setTimeout(function(){ window.location = proxyPath + "v/products"; }, 1000);            
    }).fail(function(jqXHR, textStatus, errorThrown) {
        handleHTTPError(jqXHR.statusText, jqXHR.status)
    });
}
function savePerson() {
    var form = $("#person");
    if (! form.valid()) {
        return;
    };

    var person_id = $("input#person_id").val(),
        person_email = $("input#person_email").val(),
        entities = $('select#entities').select2('data'),
        permissions = $("input[type=radio]:checked"),
        ajax_url = proxyPath + "people",
        ajax_method = "POST",
        ajax_callback = createCallBack,
        data = {};

        if ($("form#person input#person_id").length) {
            ajax_url = proxyPath + "people/" + person_id
            ajax_method = "PUT"
            ajax_callback = updateCallBack
        }

        $.each(permissions, function( index, e ) {
            data["permissions." + index +".permission_perm_name"] = $(e).attr("perm_name");
            data["permissions." + index +".permission_item_name"] = $(e).attr("item_name");
            data["permissions." + index +".permission_entity_id"] = $(e).attr("entity_id");
        });               
        $.each(entities, function( index, e ) {
            data["entities." + index +".entity_id"] = e.id;
            data["entities." + index +".entity_name"] = e.text;
        });
        $.extend(data, {
                "person_id": person_id,
                "person_email": person_email,
        });
        $.ajax({
            url: ajax_url,
            method: ajax_method,
            dataType: 'json',
            data: data,
        }).done(ajax_callback).fail(function(jqXHR, textStatus, errorThrown) {
            handleHTTPError(jqXHR.statusText, jqXHR.status)
        });  
    }