Platform
========

As I write simple libraries useful for web application development, I will
add them to this repository once I feel satisfied with their API decisions.

Currently, there is only one package under here, but I would like to add more.

platform/router
---------------

An inteface based restful router for Go. A new controller interface is created
for each request, so you can save request specific variables on the controller,
embed a mixin like LoggedInFilter into another controller, and various other 
tasks. Currently the implementation is basic, like you cannot define non-restful
routes through a nice API (since I haven't needed it yet), but I do have an idea
of how to do it.

