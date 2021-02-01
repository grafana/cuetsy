# cuetsy

A (highly experimental!) exporter/generator that can take CUE inputs and produce equivalent TypeScript objects.

CUE generally allows for the definition of canonical data specifications that can easily be used to validate data in "backend"-type contexts (e.g. devops, HTTP servers).  Cuetsy aims to extend this reach to the frontend by generating TypeScript objects (e.g. `type`, `interface`, `enum`) that are logically equivalent to their CUE counterparts. One source of data/config truth, all lifecycle stages.