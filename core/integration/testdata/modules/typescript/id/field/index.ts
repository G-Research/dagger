import { object, func } from "@github.com/G-Research/dagger";

@object()
export class Test {
  @func()
  fn(): CustomObject {
    return new CustomObject("NOOOO!!!!");
  }
}

@object()
export class CustomObject {
  @func()
  ID: string;

  constructor(id: string) {
    this.ID = id;
  }
}
