import { object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  greeting: string
  name: string

  constructor(greeting = "Hello", name = "World") {
    this.greeting = greeting
    this.name = name
  }

  @func()
  message(): string {
    return `${this.greeting} ${this.name}`
  }
}
