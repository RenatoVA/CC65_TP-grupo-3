package catalog

type Dataset struct {
	ID           string
	Label        string
	Category     string
	Years        []int
	ResourceURLs []string
}

func DefaultCatalog() []Dataset {
	return []Dataset{
		{
			ID:       "indecopi-ops3-expedientes-resueltos",
			Label:    "INDECOPI OPS3 Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS3_ExpedientesResueltos_2015.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS3_ExpedientesResueltos_2016.xls",
			},
		},
		{
			ID:       "indecopi-din-expedientes-presentados",
			Label:    "INDECOPI DIN Expedientes Presentados",
			Category: "presentados",
			Years:    []int{1993, 2018},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DIN_ExpedientesPresentados_1993_2018.xlsx",
			},
		},
		{
			ID:       "indecopi-ops1-expedientes-presentados",
			Label:    "INDECOPI OPS1 Expedientes Presentados",
			Category: "presentados",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS1_ExpedientesPresentados_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS1_ExpedientesPresentados_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-ops1-expedientes-resueltos",
			Label:    "INDECOPI OPS1 Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS1_ExpedientesResueltos_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS1_ExpedientesResueltos_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-ops3-expedientes-presentados",
			Label:    "INDECOPI OPS3 Expedientes Presentados",
			Category: "presentados",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS3_ExpedientesPresentados_2015.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS3_ExpedientesPresentados_2016.xls",
			},
		},
		{
			ID:       "indecopi-cc3-expedientes-presentados",
			Label:    "INDECOPI CC3 Expedientes Presentados",
			Category: "presentados",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC3_ExpedientesPresentados_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC3_ExpedientesPresentados_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-cc3-expedientes-resueltos",
			Label:    "INDECOPI CC3 Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC3_ExpedientesResueltos_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC3_ExpedientesResueltos_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-din-expedientes-resueltos",
			Label:    "INDECOPI DIN Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{1993, 2018},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DIN_ExpedientesResueltos_1993_2018.xlsx",
			},
		},
		{
			ID:       "indecopi-cc2-expedientes-presentados",
			Label:    "INDECOPI CC2 Expedientes Presentados",
			Category: "presentados",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC2_ExpedientesPresentados_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC2_ExpedientesPresentados_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-cc1-expedientes-presentados",
			Label:    "INDECOPI CC1 Expedientes Presentados",
			Category: "presentados",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC1_ExpedientesPresentados_2015_0.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC1_ExpedientesPresentados_2016_0.xlsx",
			},
		},
		{
			ID:       "indecopi-cc1-expedientes-resueltos",
			Label:    "INDECOPI CC1 Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC1_ExpedientesResueltos_2015_0.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC1_ExpedientesResueltos_2016_0.xlsx",
			},
		},
		{
			ID:       "indecopi-spc-expedientes-presentados",
			Label:    "INDECOPI SPC Expedientes Presentados",
			Category: "presentados",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_SPC_ExpedientesPresentados_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_SPC_ExpedientesPresentados_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-spc-expedientes-resueltos",
			Label:    "INDECOPI SPC Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_SPC_ExpedientesResueltos_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_SPC_ExpedientesResueltos_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-clc-expedientes-resueltos",
			Label:    "INDECOPI CLC Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2008.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2009.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2010.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2011.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2012.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2013.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2014.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2015.xls",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CLC_ExpedientesResueltos_2016.xls",
			},
		},
		{
			ID:       "indecopi-ops2-expedientes-presentados",
			Label:    "INDECOPI OPS2 Expedientes Presentados",
			Category: "presentados",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS2_ExpedientesPresentados_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS2_ExpedientesPresentados_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-ops2-expedientes-resueltos",
			Label:    "INDECOPI OPS2 Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS2_ExpedientesResueltos_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_OPS2_ExpedientesResueltos_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-dda-expedientes-de-infracciones-resueltos",
			Label:    "INDECOPI DDA Expedientes de Infracciones Resueltos",
			Category: "resueltos",
			Years:    []int{2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DDA_ExpedientesResueltosInfracciones_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-dda-expedientes-de-registro-resueltos",
			Label:    "INDECOPI DDA Expedientes de Registro Resueltos",
			Category: "resueltos",
			Years:    []int{2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DDA_ExpedientesResueltosRegistros_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-dsd-expedientes-otorgados",
			Label:    "INDECOPI DSD Expedientes Otorgados",
			Category: "otorgados",
			Years:    []int{2014, 2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DSD_ExpedientesOtorgados_2014.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DSD_ExpedientesOtorgados_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DSD_ExpedientesOtorgados_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-dsd-expedientes-presentados",
			Label:    "INDECOPI DSD Expedientes Presentados",
			Category: "presentados",
			Years:    []int{2014, 2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI-DSD_ExpedientesPresentados_2014.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DSD_ExpedientesPresentados_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DSD_ExpedientesPresentados_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-dsd-expedientes-resueltos",
			Label:    "INDECOPI DSD Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2014, 2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DSD_ExpedientesResueltos_2014.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DSD_ExpedientesResueltos_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_DSD_ExpedientesResueltos_2016.xlsx",
			},
		},
		{
			ID:       "indecopi-cc2-expedientes-resueltos",
			Label:    "INDECOPI CC2 Expedientes Resueltos",
			Category: "resueltos",
			Years:    []int{2015, 2016},
			ResourceURLs: []string{
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC2_ExpedientesResueltos_2015.xlsx",
				"https://www.datosabiertos.gob.pe/sites/default/files/INDECOPI_CC2_ExpedientesResueltos_2016.xlsx",
			},
		},
	}
}
