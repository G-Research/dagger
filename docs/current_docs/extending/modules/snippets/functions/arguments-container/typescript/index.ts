import { Container, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  async osInfo(ctr: Container): Promise<string> {
    return ctr.withExec(["uname", "-a"]).stdout()
  }
}
