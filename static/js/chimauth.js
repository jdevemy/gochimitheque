// handle HTTP errors
function handleHTTPError(msgText, msgStatus) {
    //console.log(msgStatus);
    switch(msgStatus) {
    case 401:
        global.displayMessage(msgText, "danger");
        break;
    case 403:
        global.displayMessage(msgText, "danger");
        break;
    case 500:
        global.displayMessage(msgText, "danger");
        break;
    default:
        global.displayMessage(msgText, "light");
        break;
    }
}

// get a captcha to resolve
function getCaptcha() {
    var email = $("#person_email").val();
    if (email == "") {
        global.displayMessage("enter your email in the login form", "warning");
    } else {
        global.displayMessage("are you a robot?", "success");
        $("#getcaptcha").fadeOut("slow");
        $.ajax({
            url: proxyPath + "captcha",
            method: 'GET',
        }).done(function(data) {
            $("input#captcha_uid").val(data.uid);
            $("img#captcha-img").attr("src", "data:image/png;base64," + data.image);
            $("div#captcha-row").removeClass("invisible");
        }).fail(function(jqXHR, textStatus, errorThrown) {
            handleHTTPError(jqXHR.responseText, jqXHR.status)
        });
    }
}

// send a password initialization link
function resetPassword() {
    var email = $("#person_email").val(),
        captcha = $("input#captcha_text").val(),
        uid = $("input#captcha_uid").val();
    if (email == "") {
        global.displayMessage("enter your email in the login form", "warning");
    } else {
        global.displayMessage("validating...", "success");
        $("#resetpassword").fadeOut("slow");
        $.ajax({
            url: proxyPath + "reset-password",
            method: 'POST',
            data: {
                person_email: email,
                captcha_text: captcha,
                captcha_uid: uid,
            }
        }).done(function(token) {
            global.displayMessage("a reinitialization link has been sent to " + email, "success");
        }).fail(function(jqXHR, textStatus, errorThrown) {
            handleHTTPError(jqXHR.responseText, jqXHR.status)
        });
        $("#person_email").val("");
        $("input#captcha_text").val("");
        $("div#captcha-row").addClass("invisible");
    }
}

// get a JWT token for the user
function getToken() {
    var email = $("#person_email").val(),
        password = $("#person_password").val();

    if (email == "") {
        global.displayMessage("enter your email in the login form", "warning");
    } else {
        $.ajax({
            url: proxyPath + "get-token",
            method: 'POST',
            data: {
                person_email: email,
                person_password: password
            }
        }).done(function(token) {
            //console.log(token);
            // store in web storage
            //window.localStorage.setItem('token', token);
            window.location.replace(proxyPath + "v/products");
        }).fail(function(jqXHR, textStatus, errorThrown) {
            handleHTTPError(jqXHR.responseText, jqXHR.status)
        });
    }
}

// returns the cookie value
function readCookie(name) {
    var nameEQ = name + "=";
    var ca = document.cookie.split(';');
    for(var i=0;i < ca.length;i++) {
        var c = ca[i];
        while (c.charAt(0)==' ') c = c.substring(1,c.length);
        if (c.indexOf(nameEQ) == 0) return c.substring(nameEQ.length,c.length);
    }
    return null;
}

function hasPermission(url, method, itemId) {
    return $.ajax({
        url: url,
        itemId: itemId,
        method: method,
    });
}

// jwt in web storage
//$.ajaxPrefilter(function( options ) {
//    options.beforeSend = function (xhr) {
//        xhr.setRequestHeader('Authorization', 'Bearer '+localStorage.getItem('token'));
//    }
//});
