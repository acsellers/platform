Platform
========

As I write simple libraries useful for web application development, I will
add them to this repository once I feel satisfied with the API.

platform/router
---------------

An inteface based restful router for Go. A new controller interface is created
for each request, so you can save request specific variables on the controller,
embed a mixin like LoggedInFilter into another controller, and various other 
tasks. You can define a Restful controller with the basic action, Index, Show, 
New, Create, Edit, Update, Delete. You can define extra Item actions or Base actions.
An Item action would be like /posts/123/share, while a Base action would be like
/posts/search. Also you can define arbitrary actions on the router using common 
actions like GET, POST, etc., then set the function to call to either be a controller
function or HandlerFunc.

In the near future, I need to work on more ways to have callable functions (like http.Handler),
a ToJavascript option for the RouteList function, better RouteList names, StripPrefix code
somewhere around Module Mounts, Params helpers, Format helpers (JSON, XML, HTML, JS).
Still plenty of things to work on.

platform/controllers
--------------------

As I write applications using platform/router, I run across similar code. After a 
few iterations of the same basic code, I extract the code out and put it in the
controllers subpackage. Currently there are 3 controller-ish things in p/c.

* ResetController - Embeddable struct to conflict with Restful actions on another
embedded struct. Doesn't conflict with PreFilter or PreItem. More of a Proof of
Concept than broadly useful.

* RenderableCtrl - Alternative to router.BaseController that provides HTML Rendering
using the multitemplate library.

* AssetController - Serve the files out of a folder, will set Cache-Control if given
a time.Duration. Uses http.ServeFile for the sending.

* AssetModule - Allows you to serve up a "public" type folder with sub-folders for css,
js, img, fonts, etc. quickly. It just builds AssetController instances as necessary.


Future Packages
===============

platform/template
-----------------

I've written multi-template, which is a bunch of HTML helpers, language parsers, 
and core functions for the html/template library. In general, it work, but 
because it is a complex problem, there are things that I did wrong. I want to
migrate the parsers to a new paradigm, document all the language features, and
make breaking API changes. Whenever I get to that, it will be platform/template.
Note that I do use multi-template for real things now, but there are bugs that
I work around.

platform/stylesheet
-------------------

I have sassy, which is a libsass binding for Go. While it has some features, I
would like to clean it up into a better library and get better documentation 
and features around it so it works better. It should also be usable for 
compressing/concatenating css stylesheets automatically then.

