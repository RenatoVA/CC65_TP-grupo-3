package main

import (
	"fmt"

	"tp-programacion-concurente/internal/excel"
)

func main() {
	files := []string{
		"data/raw/indecopi-ops2-expedientes-presentados/INDECOPI_OPS2_ExpedientesPresentados_2015.xlsx",
		"data/raw/indecopi-spc-expedientes-presentados/INDECOPI_SPC_ExpedientesPresentados_2015.xlsx",
		"data/raw/indecopi-cc1-expedientes-presentados/INDECOPI_CC1_ExpedientesPresentados_2015_0.xlsx",
	}

	for _, f := range files {
		fmt.Println("\nFILE", f)
		sheets, err := excel.ReadFile(f)
		if err != nil {
			panic(err)
		}
		for _, s := range sheets {
			fmt.Println("SHEET", s.Name, "rows", len(s.Rows), "header", s.Header)
			for i := 0; i < 3 && i < len(s.Rows); i++ {
				fmt.Println(s.Rows[i])
			}
		}
		fmt.Println("RAW PREVIEW NOT AVAILABLE THROUGH CURRENT API")
	}
}
