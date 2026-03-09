import {
  dag,
  object,
  argument,
  func,
  Directory,
  Container,
} from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  async foo(
    @argument({ ignore: ["*", "!**/*.ts"] }) source: Directory,
  ): Promise<Container> {
    return await dag
      .container()
      .from("alpine:latest")
      .withDirectory("/src", source)
      .sync()
  }
}
