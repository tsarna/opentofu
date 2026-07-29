symbols "@#$%^&*" {
    source = "./foo"
} 

symbols "nullsrc" {
    source = null
}

symbols "emptysrc" {
    source = ""
}

symbols "notstatic" {
    source = var.symsrc
}

symbols "func" {
    source = foo()
}

symbols "nsnotstatic" {
    source = "./foo"
    namespace = var.ns
}

symbols "nsfunc" {
    source = "./foo"
    namespace = foo()
}

symbols "nosource" {
    namespace = "blah"
}

symbols "nonstrsource" {
    source = [4,8,15,16,23,42]
}

symbols "nonstrns" {
    source = "./foo"
    namespace = [4,8,15,16,23,42]
}

symbols "extra" {
    source = "./foo"
    prime = 73
}

symbols "bare_src" {
    source = "abc"
}

symbols "remote" {
    source = "git::https://github.com/foo/bar"
}