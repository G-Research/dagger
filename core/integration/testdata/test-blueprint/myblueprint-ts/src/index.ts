import { dag, object, func } from "@github.com/G-Research/dagger";

@object()
export class MyblueprintTs {
  /**
   * echoes 'hello from blueprint'
   */
  @func()
  hello(): Promise<string> {
    return dag
      .container()
      .from("alpine:latest")
      .withExec(["echo", "hello from blueprint"])
      .stdout();
  }
}
