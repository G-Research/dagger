import type { float } from "@github.com/G-Research/dagger"
import { object, func } from "@github.com/G-Research/dagger"

@object()
export class MyModule {
  @func()
  addFloat(a: float, b: float): float {
    return a + b
  }
}
