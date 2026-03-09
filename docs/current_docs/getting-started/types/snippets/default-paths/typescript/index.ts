import { Directory, File, argument, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  async readDir(
    @argument({ defaultPath: "/" }) source: Directory,
  ): Promise<string[]> {
    return await source.entries()
  }
}
