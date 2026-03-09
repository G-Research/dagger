import { object, func } from "@github.com/G-Research/dagger"

@object()
export class MyModule {
  @func()
  addInteger(a: number, b: number): number {
    return a + b
  }
}
