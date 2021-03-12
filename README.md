<p align="center">
  <img
    width="300"
    src="https://raw.githubusercontent.com/sdboyer/cuetsy/main/docs/logo/cuetsy.svg"
    alt="Cuetsy Logo"
  />
</p>

<p align="center">
  <a href="https://cloud.drone.io/sdboyer/cuetsy">
    <img src="https://img.shields.io/drone/build/sdboyer/cuetsy?style=flat-square">
  </a>
  <a href="https://github.com/sdboyer/cuetsy/releases">
    <img src="https://img.shields.io/github/release/sdboyer/cuetsy?style=flat-square" />
  </a>
  <img src="https://img.shields.io/github/contributors/sdboyer/cuetsy?style=flat-square" />
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
DiceFaces: 1 | 2 | 3 | 4 | 5 | 6 @cuetsy(targetType="type")

Animal: {
    Name: string
    Sound: string
} @cuetsy(targetType="interface")

LeggedAnimal: Animal & {
    Legs: int
} @cuetsy(targetType="interface")

Pets: "Cat" | "Dog" | "Horse" @cuetsy(targetType="enum")
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
$ go install github.com/sdboyer/cuetsy/cmd/cuetsy
```

## Usage

`cuetsy` must be invoked on files as follows:

```shell
$ cuetsy [file.cue]
```

This will create a logically equivalent `[file].ts`

### Union Types

| CUE                                                                        | TypeScript                                                                                   | `@cuetsy(targetType)` |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | --------------------- |
| [Disjunction](https://cuelang.org/docs/tutorials/tour/types/disjunctions/) | [Union Type](https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#union-types) | `type`                |

Union types are expressed in CUE and TypeScript nearly the same way, namely a
series of disjunctions (`a | b | c`):

<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
MyUnion: 1 | 2 | 3 | 4 | 5 | 6 @cuetsy(targetType="type")
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

| CUE                                                               | TypeScript                                                                                 | `@cuetsy(targetType)` |
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
} @cuetsy(targetType="interface")
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
} @cuetsy(targetType="interface")

BInterface: AInterface & {
    BField: int
} @cuetsy(targetType="interface")
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

| CUE                                                                                                                                           | TypeScript                                                                       | `@cuetsy(targetType)` |
| --------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | --------------------- |
| [Disjunction](https://cuelang.org/docs/tutorials/tour/types/disjunctions/), [Struct](https://cuelang.org/docs/tutorials/tour/types/optional/) | [Enum](https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#enums) | `enum`                |

Cuetsy supports two ways of expressing TypeScript enums in CUE:

#### Disjunction style

Disjunctions may be used in a very similar way to Union Types. The keys of the
enum are automatically inferred as the titled camel-case variant of their value:

<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
MyEnum: "foo" | "bar" | "baz" @cuetsy(targetType="enum")
```

</td>
<td>

```typescript
export enum MyEnum {
  Bar = "bar",
  Baz = "baz",
  Foo = "foo",
}
```

</td>
</tr>
</table>

#### Struct style

If the implicit keys are insufficient, the `struct` based style gives more
control:

<table>
<tr><th>CUE</th><th>TypeScript</th></tr>
<tr>
<td>

```cue
MyEnum: {
    Foo: "foo"
    iCanChoose: "whateverILike"
} @cuetsy(targetType="enum")
```

</td>
<td>

```typescript
export enum MyEnum {
  Foo = "foo",
  iCanChoose = "whateverILike",
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
MyUnion: 1 | 2 | *3 @cuetsy(targetType="type")

MyDisjEnum: "foo" | *"bar" @cuetsy(targetType="enum")
MyStructEnum: {
    A: "Foo"
    B: "Bar" @cuetsy(enumDefault)
} @cuetsy(targetType="enum")

MyInterface: {
    num: int | *6
    txt: string | *"CUE"
    enm: MyDisjEnum
} @cuetsy(targetType="interface")
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
