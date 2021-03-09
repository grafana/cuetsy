package cuetsy

E1: "e1str1" | "e1str2" | "e1str3" @cuetsy(targetType="enum")
E2: "e2str1" | "e2str2" | "e2str3" | "e2str4" @cuetsy(targetType="enum")
E3: {
  Walla: "laadeedaa"
  run: "OMG"
} @cuetsy(targetType="enum")

I1: {
  I1_OptionalDisjunctionLiteral?: "other" | "values" | 2
  I1_FloatLiteral: 4.4
  I1_Top: _
} @cuetsy(targetType="interface")

I2: {
  I2_Number: number
  I2_OptionalInterfaceReference?: I1
  I2_OptionalBool?: bool
  I2_TypedList: [...number]
} @cuetsy(targetType="interface")

I3: {
  I3_EnumReference: E1
  I3_OptionalString?: string
  I3_OptionalNumber?: number
} @cuetsy(targetType="interface")

I4: I2 & I3 & {
  I4_OptionalEnumReference?: E2
} @cuetsy(targetType="interface")

I5: I2 & {
  I5_OptionalEnumReference?: E2
}  & I3 @cuetsy(targetType="interface")

I6: I2 & I3 @cuetsy(targetType="interface")
