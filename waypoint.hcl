# The name of your project. A project typically maps 1:1 to a VCS repository.
# This name must be unique for your Waypoint server. If you're running in
# local mode, this must be unique to your machine.
project = "win-loss-api"

# Labels can be specified for organizational purposes.
# labels = { "foo" = "bar" }

# An application to deploy.
app "web" {
    labels = {
        "service" = "win-loss-api",
        "env" = "dev",
    }

    build {
        use "pack" {}

        registry {
            use "docker" {
                image = "ghcr.io/r35krag0th/win-loss-api"
                tag = "latest"
            }
        }

    }

    # Deploy to Docker
    deploy {
        use "docker" {}
    }
}
