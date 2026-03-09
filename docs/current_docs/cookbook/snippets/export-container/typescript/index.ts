import { dag, Container, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  /**
   * Return a container
   */
  @func()
  base(): Container {
    return dag
      .container()
      .from("alpine:latest")
      .withExec(["mkdir", "/src"])
      .withExec(["touch", "/src/foo", "/src/bar"])
  }
}
