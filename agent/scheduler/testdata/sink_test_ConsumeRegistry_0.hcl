pod "first" {
  constraint {
    "${meta.first}" = "true"
  }
}

pod "second" {
  constraint {
    "${meta.second}" = "true"
  }
}

pod "third" {
  constraint {
    "${meta.first}" = "true"
  }
  resource "port" "8080" {

  }
}
