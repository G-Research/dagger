import { object, func } from "@github.com/G-Research/dagger";

@object()
export class Test {
  @func()
  fn(id: string): string {
    return id;
  }
}
