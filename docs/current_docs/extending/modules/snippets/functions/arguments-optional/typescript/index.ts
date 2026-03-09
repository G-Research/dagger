import { object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  hello(name?: string): string {
    if (name) {
      return `Hello, ${name}`
    }
    return "Hello, world"
  }
}
