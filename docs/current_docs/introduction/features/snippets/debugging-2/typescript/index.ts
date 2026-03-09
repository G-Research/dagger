import { dag, object, Directory, Container, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  async foo(): Container {
    return await dag
      .container()
      .from("alpine:latest")
      .terminal()
      .withExec(["sh", "-c", "echo hello world > /foo"])
      .terminal()
  }
}
