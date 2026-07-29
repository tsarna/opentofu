symbols "fully_overridden" {
  source = "./foo"
  namespace = "bar"
}

symbols "partially_overridden" {
  source = "./qux"
}

symbols "introduce_ns" {
  source = "./xyz"
}

symbols "ns_not_clobbered" {
  source = "./s"
  namespace = "ns"
}
