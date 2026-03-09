import { dag, Directory, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  example(buildSrc: Directory, buildArgs: string[]): Directory {
    return dag.golang().build({ source: buildSrc, args: buildArgs }).terminal()
  }
}
