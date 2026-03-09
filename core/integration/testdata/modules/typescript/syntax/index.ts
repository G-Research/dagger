import { object, func } from "@github.com/G-Research/dagger"

@object()
export class Syntax {
	@func()
	singleQuoteDefaultArgHello(msg: string = 'world'): string {
		return `hello ${msg}`
	}

	@func()
	doubleQuotesDefaultArgHello(msg: string = "world"): string {
		return `hello ${msg}`
	}
}