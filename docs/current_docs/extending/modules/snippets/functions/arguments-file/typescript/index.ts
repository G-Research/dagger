import { dag, object, func, File } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  async readFile(source: File): Promise<string> {
    return await dag
      .container()
      .from("alpine:latest")
      .withFile("/src/myfile", source)
      .withExec(["cat", "/src/myfile"])
      .stdout()
  }
}
