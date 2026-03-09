import { object, func } from "@github.com/G-Research/dagger"

@object()
export class Test {
  @func()
  id(): string {
    return "NOOOO!!!!"
  }
}
