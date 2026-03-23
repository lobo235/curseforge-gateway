job "curseforge-gateway" {
  node_pool   = "default"
  datacenters = ["dc1"]
  type        = "service"

  group "curseforge-gateway" {
    count = 1

    network {
      port "http" {
        to = 8080
      }
    }

    service {
      name     = "curseforge-gateway"
      port     = "http"
      provider = "consul"
      tags = [
        "traefik.enable=true",
        "traefik.http.routers.curseforge-gateway.rule=Host(`curseforge-gateway.example.com`)",
        "traefik.http.routers.curseforge-gateway.entrypoints=websecure",
        "traefik.http.routers.curseforge-gateway.tls=true",
      ]

      check {
        type     = "http"
        path     = "/health"
        port     = "http"
        interval = "30s"
        timeout  = "5s"

        check_restart {
          limit = 3
          grace = "30s"
        }
      }
    }

    restart {
      attempts = 3
      interval = "2m"
      delay    = "15s"
      mode     = "fail"
    }

    vault {
      cluster     = "default"
      change_mode = "restart"
    }

    task "curseforge-gateway" {
      driver = "docker"

      config {
        image = "ghcr.io/lobo235/curseforge-gateway:latest"
        ports = ["http"]
      }

      template {
        data = <<EOF
{{ with secret "kv/data/nomad/default/curseforge-gateway" }}
CF_API_KEY={{ .Data.data.cf_api_key }}
GATEWAY_API_KEY={{ .Data.data.gateway_api_key }}
{{ end }}
EOF
        destination = "secrets/curseforge-gateway.env"
        env         = true
      }

      env {
        PORT      = "8080"
        LOG_LEVEL = "info"
      }

      resources {
        cpu    = 100
        memory = 64
      }

      kill_timeout = "35s"
    }
  }
}
