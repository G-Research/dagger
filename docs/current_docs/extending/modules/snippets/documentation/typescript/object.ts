import { object } from "@github.com/G-Research/dagger"

/**
 * The object represents a single user of the system.
 */
@object()
class MyModule {
  @func()
  name: string
  @func()
  age: number

  constructor(
    /**
     * The name of the user.
     */
    name: string,
    /**
     * The age of the user.
     */
    age: number,
  ) {
    this.name = name
    this.age = age
  }
}
