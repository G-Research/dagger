import { dag, Container, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  container(): Container {
    return dag
      .container()
      .from("alpine:latest")
      .terminal()
      .withExec(["sh", "-c", "echo hello world > /foo && cat /foo"])
      .terminal()
  }
}
