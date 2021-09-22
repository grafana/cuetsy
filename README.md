<p align="center">
  <img
    width="300"
    src="https://raw.githubusercontent.com/grafana/cuetsy/main/docs/logo/cuetsy.svg"
    alt="Cuetsy Logo"
  />
</p>

<p align="center">
  <a href="https://drone.grafana.net/grafana/cuetsy">
    <img src="https://img.shields.io/drone/build/grafana/cuetsy?style=flat-square">
  </a>
  <a href="https://github.com/grafana/cuetsy/releases">
    <img src="https://img.shields.io/github/release/grafana/cuetsy?style=flat-square" />
  </a>
  <img src="https://img.shields.io/github/contributors/grafana/cuetsy?style=flat-square" />
</p>

<p align="center">
  <a href="#installation">Installation</a>
  ·
  <a href="#usage">Usage</a>
</p>

<h1>cue<i>ts</i>y</h1>

**Converting CUE objects to their TypeScript equivalent** _(highly experimental!)_

- [**CUE**](https://cuelang.org) makes defining and validating canonical data
  specification easy
- [**TypeScript**](https://typescript.com) is dominant in the frontend, but
  cannot natively benefit from this
- [**CUE types**](https://cuelang.org/docs/tutorials/tour/types/) have direct
  **TypeScript equivalents**, so cuetsy can bridge this gap

### Example

<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
DiceFaces: 1 | 2 | 3 | 4 | 5 | 6 @cuetsy(kind="type")

Animal: {
    Name: string
    Sound: string
} @cuetsy(kind="interface")

LeggedAnimal: Animal & {
    Legs: int
} @cuetsy(kind="interface")

Pets: "Cat" | "Dog" | "Horse" @cuetsy(kind="enum")
```

</td>
<td>

```typescript
export type DiceFaces = 1 | 2 | 3 | 4 | 5 | 6;
export interface Animal {
  Name: string;
  Sound: string;
}
export interface LeggedAnimal extends Animal {
  Legs: number;
}
export enum Pets {
  Cat = "Cat",
  Dog = "Dog",
  Horse = "Horse",
}
```

</td>
</tr>
</table>

### Status

Cuetsy is in its early development, so it does not support all TypeScript
features. However, the following are supported:

- **Types**
  - **[Unions](#union-types)**
  - **[Interfaces](#interfaces)**
  - **[Enums](#enums)**
- **[Default `const`](#defaults)**

## Installation

Cuetsy can be installed using [Go](https://golang.org) 1.16+

```shell
$ go install github.com/grafana/cuetsy/cmd/cuetsy
```

## Usage

`cuetsy` must be invoked on files as follows:

```shell
$ cuetsy [file.cue]
```

This will create a logically equivalent `[file].ts`

### Union Types

| CUE                                                                        | TypeScript                                                                                   | `@cuetsy(kind)` |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | --------------------- |
| [Disjunction](https://cuelang.org/docs/tutorials/tour/types/disjunctions/) | [Union Type](https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#union-types) | `type`                |

Union types are expressed in CUE and TypeScript nearly the same way, namely a
series of disjunctions (`a | b | c`):

<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
MyUnion: 1 | 2 | 3 | 4 | 5 | 6 @cuetsy(kind="type")
```

</td>
<td>

```typescript
export type MyUnion = 1 | 2 | 3 | 4 | 5 | 6;
```

</td>
</tr>
</table>

### Interfaces

| CUE                                                               | TypeScript                                                                                 | `@cuetsy(kind)` |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | --------------------- |
| [Struct](https://cuelang.org/docs/tutorials/tour/types/optional/) | [Interface](https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#interfaces) | `interface`           |

TypeScript interfaces are expressed as regular structs in CUE.

**Caveats:**

- Nested structs are not supported

<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
MyInterface: {
    Num: number
    Text: string
    List: [...number]
    Truth: bool
} @cuetsy(kind="interface")
```

</td>
<td>

```typescript
export interface MyInterface {
  List: number[];
  Num: number;
  Text: string;
  Truth: boolean;
}
```

</td>
</tr>
</table>

#### Inheritance

Interfaces can optionally inherit from another interface. This is expressed
using the union operator `&`:

<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
AInterface: {
    AField: string
} @cuetsy(kind="interface")

BInterface: AInterface & {
    BField: int
} @cuetsy(kind="interface")
```

</td>
<td>

```typescript
export interface AInterface {
  AField: string;
}
export interface BInterface extends AInterface {
  BField: number;
}
```

</td>
</tr>
</table>

### Enums

| CUE                                                                                                                                           | TypeScript                                                                       | `@cuetsy(kind)` |
| --------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | --------------------- |
| [Disjunction](https://cuelang.org/docs/tutorials/tour/types/disjunctions/), [Struct](https://cuelang.org/docs/tutorials/tour/types/optional/) | [Enum](https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#enums) | `enum`                |

TypeScript's enums are union types, and are a mostly-exact mapping of what can
be expressed with CUE's disjunctions. Disjunctions may contain only string or
numeric values.

The member names (keys) of the TypeScript enum are automatically inferred as the
titled camel-case variant of their string value, but may be explicitly specified
using the `memberNames` attribute. If the disjunction contains any numeric
values, `memberNames` must be specified.

<table>
<tr>
<th>CUE</th>
<th>TypeScript</th>
</tr>
<tr>
<td>

```cue
// Enum-level comment
// Foo: member-level comment
// Bar: member-level comment
AutoCamel: "foo" | "bar" @cuetsy(kind="enum")
// Enum-level comment
// Foo: member-level comment
// Bar: member-level comment
ManualCamel: "foo" | "bar" @cuetsy(kind="enum",memberNames="Foo|Bar")
Arbitrary: "foo" | "bar" @cuetsy(kind="enum",memberNames="Zip|Zap")
Numeric: 0 | 1 | 2 @cuetsy(kind="enum",memberNames="Zero|One|Two")
ErrMismatchLen: "a" | "b" | "c" @cuetsy(kind="enum",memberNames="a|b")
ErrNamelessNumerics: 0 | 1 | 2 @cuetsy(kind="enum")
```

</td>
<td>

```ts
/**
 * Enum-level comment
 **/
enum AutoCamel {
  // member-level comment
  Foo = "foo",
  // member-level comment
  Bar = "bar",
}
/**
 * Enum-level comment
 **/
enum ManualCamel {
  // member-level comment
  Foo = "foo",
  // member-level comment
  Bar = "bar",
}
enum Arbitrary {
  Zip = "foo",
  Zap = "bar",
}
enum Numeric {
  Zero = 0,
  One = 1,
  Two = 2,
}
```
</td>
</tr>
</table>

### Defaults

| CUE                                                                 | TypeScript                                                                                    |
| ------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| [Defaults](https://cuelang.org/docs/tutorials/tour/types/defaults/) | [`const`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/const) |

Cuetsy can optionally generate a `const` for each type that holds default
values. For that, attach [CUE Default
Values](https://cuelang.org/docs/tutorials/tour/types/defaults/) to your type
definitions:

<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
MyUnion: 1 | 2 | *3 @cuetsy(kind="type")

MyDisjEnum: "foo" | *"bar" @cuetsy(kind="enum")
MyStructEnum: {
    A: "Foo"
    B: "Bar" @cuetsy(enumDefault)
} @cuetsy(kind="enum")

MyInterface: {
    num: int | *6
    txt: string | *"CUE"
    enm: MyDisjEnum
} @cuetsy(kind="interface")
```

</td>
<td>

```typescript
export type MyUnion = 1 | 2 | 3;
export const myUnionDefault: MyUnion = 3;

export enum MyDisjEnum {
  Bar = "bar",
  Foo = "foo",
}
export const myDisjEnumDefault: MyDisjEnum = MyDisjEnum.Bar;

export enum MyStructEnum {
  A = "Foo",
  B = "Bar",
}
export const myStructEnumDefault: MyStructEnum = MyStructEnum.B;

export interface MyInterface {
  enm: MyDisjEnum;
  num: number;
  txt: string;
}
export const myInterfaceDefault: MyInterface = {
  enm: myDisjEnumDefault,
  num: 6,
  txt: "CUE",
};
```

</td>
</tr>
</table>
