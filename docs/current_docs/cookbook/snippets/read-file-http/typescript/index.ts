import { dag, object, func, Container } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  readFileHttp(url: string): Container {
    const file = dag.http(url)
    return dag.container().from("alpine:latest").withFile("/src/myfile", file)
  }
}
