import { dag, Container, Directory, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  /**
   * Return a container with a mounted directory
   */
  @func()
  mountDirectory(
    /**
     * Source directory
     */
    source: Directory,
  ): Container {
    return dag
      .container()
      .from("alpine:latest")
      .withMountedDirectory("/src", source)
  }
}
