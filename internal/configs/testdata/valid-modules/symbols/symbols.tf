symbols "foo" {
    source = "./foo"
}

symbols "bar" {
    source = "./bar"
}

symbols "with_ns" {
    source = "./xyz"
    namespace = "baz::qux"
}

symbols "blank_ns" {
    source = "./xyz"
    namespace = ""
}

symbols "null_ns" {
    source = "./xyz/../abc"
    namespace = null
}

symbols "foo2" {
    source = "./foo"
}