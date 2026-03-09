import { object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  greeting = "Hello"

  @func()
  name = "World"

  @func()
  withGreeting(greeting: string): MyModule {
    this.greeting = greeting
    return this
  }

  @func()
  withName(name: string): MyModule {
    this.name = name
    return this
  }

  @func()
  message(): string {
    return `${this.greeting} ${this.name}`
  }
}
