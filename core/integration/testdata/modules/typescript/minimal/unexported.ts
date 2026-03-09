import { object, func } from "@github.com/G-Research/dagger"

@object()
export class Foo {
    @func()
    hello(name: string): Foo {
        return new Foo()
    }
}

@object()
class Minimal {
    @func()
    hello(name: string): string {
        return name
    } 
}

class Bar {
    @func()
    hello(name: string): string {
        return name
    }
}
