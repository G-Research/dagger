import { dag, Container, Directory, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  /**
   * Build an application using cached dependencies
   */
  @func()
  build(
    /**
     * Source code location
     */
    source: Directory,
  ): Container {
    return dag
      .container()
      .from("node:21")
      .withDirectory("/src", source)
      .withWorkdir("/src")
      .withMountedCache("/root/.npm", dag.cacheVolume("node-21"))
      .withExec(["npm", "install"])
  }
}
