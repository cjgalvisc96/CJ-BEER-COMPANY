// Atlas configuration for schema migrations.
// Apply:     task migrate:apply         (or: atlas migrate apply --env local)
// New diff:  task migrate:diff -- <name>
// Re-hash:   task migrate:hash

variable "db_url" {
  type    = string
  default = getenv("ATLAS_DB_URL")
}

env "local" {
  url = var.db_url != "" ? var.db_url : "postgres://beer:beerpassword@localhost:5432/beer?sslmode=disable"
  dev = "docker://postgres/16/dev?search_path=public"
  migration {
    dir = "file://versions"
  }
  format {
    migrate {
      apply = "{{ json . }}"
    }
  }
}

env "dev" {
  url = getenv("ATLAS_DB_URL")
  migration {
    dir = "file://versions"
  }
}

env "prod" {
  url = getenv("ATLAS_DB_URL")
  migration {
    dir = "file://versions"
  }
}
