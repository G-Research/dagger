import { Container, object, func, argument } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  async version(
    @argument({ defaultAddress: "alpine:latest" })
    ctr: Container,
  ): Promise<string> {
    return ctr.withExec(["cat", "/etc/alpine-release"]).stdout()
  }
}
