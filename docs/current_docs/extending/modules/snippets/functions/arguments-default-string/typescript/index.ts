import { object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  hello(name = "world"): string {
    return `Hello, ${name}`
  }
}
