import { object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  hello(shout: boolean): string {
    const message = "Hello, world"
    if (shout) {
      return message.toUpperCase()
    }
    return message
  }
}
