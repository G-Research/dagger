import { object, func } from "@github.com/G-Research/dagger"

@object()
export class Greeter {
  greeting: string

  constructor(greeting = "Hello") {
    this.greeting = greeting
  }

  /**
   * Greets the provided name.
   */
  @func()
  hello(name: string): string {
    return `${this.greeting}, ${name}!`
  }
}
