import { dag, Container, Directory, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  /**
   * Return a container with a specified directory
   */
  @func()
  copyDirectory(
    /**
     * Source directory
     */
    source: Directory,
  ): Container {
    return dag.container().from("alpine:latest").withDirectory("/src", source)
  }
}
