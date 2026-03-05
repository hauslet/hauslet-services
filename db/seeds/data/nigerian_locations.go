package data

import "hauslet/db/seeds/utils"

// NigerianLocation represents a specific location in Nigeria
type NigerianLocation struct {
	City      string
	State     string
	Area      string
	Address   string
	PostCode  string
	Latitude  float64
	Longitude float64
	Tier      string // "premium", "mid", "budget"
}

// NigerianLocations contains realistic Nigerian property locations
var NigerianLocations = []NigerianLocation{

	// ─────────────────────────────────────────────
	// LAGOS STATE — Premium Areas
	// ─────────────────────────────────────────────
	{"Eti-Osa", "Lagos", "Victoria Island", "12 Ahmadu Bello Way", "101241", 6.4281, 3.4219, "premium"},
	{"Eti-Osa", "Lagos", "Ikoyi", "45 Bourdillon Road", "106104", 6.4550, 3.4315, "premium"},
	{"Eti-Osa", "Lagos", "Ikoyi", "7 Gerard Road", "106104", 6.4612, 3.4408, "premium"},
	{"Eti-Osa", "Lagos", "Lekki Phase 1", "23 Admiralty Way", "105102", 6.4389, 3.5271, "premium"},
	{"Eti-Osa", "Lagos", "Lekki Phase 1", "11 Fola Osibo Street", "105102", 6.4437, 3.5195, "premium"},
	{"Eti-Osa", "Lagos", "Old Ikoyi", "3 Queens Drive", "106104", 6.4631, 3.4280, "premium"},
	{"Eti-Osa", "Lagos", "Banana Island", "5 Bourdillon Court", "106104", 6.4745, 3.4512, "premium"},
	{"Eti-Osa", "Lagos", "Oniru", "18 Oniru Estate Road", "105102", 6.4302, 3.5101, "premium"},
	{"Lagos Island", "Lagos", "Lagos Island", "22 Marina Street", "101001", 6.4541, 3.3947, "premium"},

	// ─────────────────────────────────────────────
	// LAGOS STATE — Mid-tier Areas
	// ─────────────────────────────────────────────
	{"Ikeja", "Lagos", "Ikeja GRA", "18 Obafemi Awolowo Way", "100271", 6.5833, 3.3522, "mid"},
	{"Ikeja", "Lagos", "Ikeja GRA", "9 Isaac John Street", "100271", 6.5901, 3.3497, "mid"},
	{"Ikeja", "Lagos", "Allen Avenue", "34 Allen Avenue", "100271", 6.6018, 3.3515, "mid"},
	{"Ikeja", "Lagos", "Maryland", "50 Mobolaji Bank Anthony Way", "100271", 6.5701, 3.3603, "mid"},
	{"Lagos Mainland", "Lagos", "Yaba", "32 Herbert Macaulay Street", "101245", 6.5095, 3.3711, "mid"},
	{"Lagos Mainland", "Lagos", "Yaba", "14 Commercial Avenue", "101245", 6.5051, 3.3769, "mid"},
	{"Lagos Mainland", "Lagos", "Ebute Meta", "7 Oyingbo Road", "101245", 6.4906, 3.3836, "mid"},
	{"Surulere", "Lagos", "Surulere", "56 Adeniran Ogunsanya Street", "101283", 6.4969, 3.3539, "mid"},
	{"Surulere", "Lagos", "Surulere", "21 Bode Thomas Street", "101283", 6.4998, 3.3587, "mid"},
	{"Surulere", "Lagos", "Iponri", "3 Iponri Estate Road", "101283", 6.4872, 3.3651, "mid"},
	{"Apapa", "Lagos", "Apapa GRA", "10 Burma Road", "100001", 6.4493, 3.3601, "mid"},
	{"Eti-Osa", "Lagos", "Ajah", "45 Lekki-Epe Expressway", "105102", 6.4694, 3.5890, "mid"},
	{"Eti-Osa", "Lagos", "Sangotedo", "12 Orchid Road", "105102", 6.4501, 3.6102, "mid"},
	{"Kosofe", "Lagos", "Magodo Phase 2", "15 Shangisha Road", "100244", 6.6024, 3.3959, "mid"},
	{"Kosofe", "Lagos", "Magodo Phase 1", "8 Isheri Road", "100244", 6.5870, 3.3912, "mid"},

	// ─────────────────────────────────────────────
	// LAGOS STATE — Budget Areas
	// ─────────────────────────────────────────────
	{"Lagos Mainland", "Lagos", "Gbagada", "25 Gbagada Expressway", "100234", 6.5432, 3.3870, "budget"},
	{"Lagos Mainland", "Lagos", "Gbagada Phase 2", "17 Peter Odili Road", "100234", 6.5512, 3.3951, "budget"},
	{"Kosofe", "Lagos", "Ojota", "89 Ikorodu Road", "100242", 6.5892, 3.3783, "budget"},
	{"Kosofe", "Lagos", "Ketu", "61 Lagos-Ibadan Expressway", "100242", 6.6010, 3.3860, "budget"},
	{"Kosofe", "Lagos", "Mile 12", "33 Mile 12 Market Road", "100242", 6.6149, 3.3928, "budget"},
	{"Alimosho", "Lagos", "Egbeda", "14 Idimu Road", "100283", 6.5640, 3.2740, "budget"},
	{"Alimosho", "Lagos", "Ikotun", "29 Ikotun Road", "100283", 6.5301, 3.2892, "budget"},
	{"Alimosho", "Lagos", "Alimosho", "5 Shasha Road", "100283", 6.5893, 3.2601, "budget"},
	{"Mushin", "Lagos", "Mushin", "22 Mushin Road", "100009", 6.5290, 3.3512, "budget"},
	{"Agege", "Lagos", "Agege", "11 Orile Agege Road", "100109", 6.6198, 3.3248, "budget"},
	{"Ifako-Ijaiye", "Lagos", "Ogba", "37 Ogba-Agege Road", "100237", 6.6072, 3.3381, "budget"},
	{"Ojo", "Lagos", "Ojo", "16 Badagry Expressway", "102103", 6.4685, 3.2041, "budget"},
	{"Ajeromi-Ifelodun", "Lagos", "Ajegunle", "8 Ajegunle Road", "100009", 6.4627, 3.3427, "budget"},
	{"Ikorodu", "Lagos", "Ikorodu", "54 Lagos Road", "104101", 6.6194, 3.5044, "budget"},
	{"Ibeju-Lekki", "Lagos", "Awoyaya", "7 Lekki-Epe Expressway", "105101", 6.4812, 3.6892, "budget"},

	// ─────────────────────────────────────────────
	// ABUJA (FCT) — Premium Areas
	// ─────────────────────────────────────────────
	{"Abuja", "FCT", "Maitama", "10 Aguiyi Ironsi Street", "900271", 9.0820, 7.4950, "premium"},
	{"Abuja", "FCT", "Maitama", "3 Udi Hills Street", "900271", 9.0874, 7.4882, "premium"},
	{"Abuja", "FCT", "Asokoro", "5 Yakubu Gowon Crescent", "900103", 9.0330, 7.5270, "premium"},
	{"Abuja", "FCT", "Asokoro", "17 Danube Street", "900103", 9.0289, 7.5311, "premium"},
	{"Abuja", "FCT", "Central Business District", "22 Herbert Macaulay Way", "900103", 9.0575, 7.4898, "premium"},
	{"Abuja", "FCT", "Wuse 2", "14 Aminu Kano Crescent", "900288", 9.0742, 7.4867, "premium"},

	// ─────────────────────────────────────────────
	// ABUJA (FCT) — Mid-tier Areas
	// ─────────────────────────────────────────────
	{"Abuja", "FCT", "Gwarinpa", "45 6th Avenue", "900108", 9.1108, 7.4165, "mid"},
	{"Abuja", "FCT", "Gwarinpa", "12 3rd Avenue", "900108", 9.1053, 7.4219, "mid"},
	{"Abuja", "FCT", "Jabi", "12 Ladi Kwali Street", "900108", 9.0704, 7.4329, "mid"},
	{"Abuja", "FCT", "Garki 2", "9 Ralph Shodeinde Street", "900103", 9.0527, 7.4772, "mid"},
	{"Abuja", "FCT", "Wuse", "31 Adetokunbo Ademola Crescent", "900288", 9.0651, 7.4777, "mid"},
	{"Abuja", "FCT", "Utako", "20 Euphrates Street", "900108", 9.0879, 7.4525, "mid"},
	{"Abuja", "FCT", "Lugbe", "7 Airport Road", "900107", 9.0041, 7.4210, "mid"},

	// ─────────────────────────────────────────────
	// ABUJA (FCT) — Budget Areas
	// ─────────────────────────────────────────────
	{"Abuja", "FCT", "Karu", "33 Nyanya Road", "901101", 8.9981, 7.5401, "budget"},
	{"Abuja", "FCT", "Kubwa", "15 Phase 4 Kubwa", "900108", 9.1412, 7.3572, "budget"},
	{"Abuja", "FCT", "Gwagwalada", "8 Phase 1 Road", "902101", 8.9432, 7.0812, "budget"},
	{"Abuja", "FCT", "Zuba", "22 Zuba Market Road", "900106", 9.1509, 7.2119, "budget"},

	// ─────────────────────────────────────────────
	// PORT HARCOURT — Premium Areas
	// ─────────────────────────────────────────────
	{"Port Harcourt", "Rivers", "GRA Phase 2", "15 Stadium Road", "500102", 4.8156, 7.0498, "premium"},
	{"Port Harcourt", "Rivers", "GRA Phase 3", "7 Peter Odili Road", "500102", 4.8230, 7.0612, "premium"},
	{"Port Harcourt", "Rivers", "Old GRA", "4 Aggrey Road", "500102", 4.8094, 7.0412, "premium"},
	{"Port Harcourt", "Rivers", "Rumuola", "18 Rumuola Road", "500102", 4.8489, 7.0372, "premium"},

	// ─────────────────────────────────────────────
	// PORT HARCOURT — Mid / Budget
	// ─────────────────────────────────────────────
	{"Port Harcourt", "Rivers", "D-Line", "42 Aba Road", "500102", 4.8357, 7.0129, "mid"},
	{"Port Harcourt", "Rivers", "Elekahia", "11 Elekahia Road", "500102", 4.8201, 7.0282, "mid"},
	{"Port Harcourt", "Rivers", "Ada George", "25 Ada George Road", "500258", 4.8701, 7.0291, "mid"},
	{"Port Harcourt", "Rivers", "Rumuigbo", "9 Rumuigbo Road", "500258", 4.8531, 7.0101, "budget"},
	{"Port Harcourt", "Rivers", "Rumuola", "50 Ikwerre Road", "500258", 4.8623, 7.0012, "budget"},
	{"Port Harcourt", "Rivers", "Ozuoba", "17 East-West Road", "500258", 4.9012, 6.9782, "budget"},

	// ─────────────────────────────────────────────
	// IBADAN — Oyo State
	// ─────────────────────────────────────────────
	{"Ibadan North", "Oyo", "Bodija", "22 Awolowo Avenue", "200284", 7.4340, 3.9087, "mid"},
	{"Ibadan North", "Oyo", "Bodija", "5 University Road", "200284", 7.4412, 3.8991, "mid"},
	{"Ibadan North", "Oyo", "Agodi GRA", "14 Agodi Road", "200271", 7.3958, 3.9019, "premium"},
	{"Ibadan North", "Oyo", "Jericho GRA", "30 Jericho Road", "200271", 7.4012, 3.8902, "premium"},
	{"Ibadan South-West", "Oyo", "Oluyole Estate", "8 Oluyole Estate Road", "200265", 7.3691, 3.8812, "mid"},
	{"Ibadan North-East", "Oyo", "Iwo Road", "45 Iwo Road", "200212", 7.4289, 3.9312, "budget"},
	{"Ibadan South-East", "Oyo", "Iyaganku", "19 Iyaganku Crescent", "200271", 7.3890, 3.8980, "mid"},
	{"Akinyele", "Oyo", "Ojoo", "31 Ojoo Road", "200132", 7.4782, 3.9212, "budget"},
	{"Ona-Ara", "Oyo", "Awotan", "12 Awotan Road", "200131", 7.3512, 3.9812, "budget"},

	// ─────────────────────────────────────────────
	// KANO STATE
	// ─────────────────────────────────────────────
	{"Kano Municipal", "Kano", "Nassarawa GRA", "5 Ibrahim Taiwo Road", "700241", 12.0022, 8.5182, "premium"},
	{"Kano Municipal", "Kano", "GRA", "18 Zoo Road", "700241", 12.0102, 8.5301, "premium"},
	{"Kano Municipal", "Kano", "Bompai", "9 Bompai Road", "700241", 12.0341, 8.5412, "mid"},
	{"Fagge", "Kano", "Sabon Gari", "23 Murtala Mohammed Way", "700233", 12.0072, 8.5101, "mid"},
	{"Dala", "Kano", "Rijiyar Lemo", "11 Kofar Wambai Road", "700221", 11.9891, 8.5089, "budget"},
	{"Gwale", "Kano", "Gyadi-Gyadi", "7 Gyadi-Gyadi Road", "700222", 11.9951, 8.5019, "budget"},
	{"Ungogo", "Kano", "Kumbotso", "34 Challawa Road", "700271", 12.0542, 8.5501, "budget"},

	// ─────────────────────────────────────────────
	// ENUGU STATE
	// ─────────────────────────────────────────────
	{"Enugu North", "Enugu", "GRA", "16 Abakaliki Road", "400241", 6.4698, 7.5261, "premium"},
	{"Enugu North", "Enugu", "Independence Layout", "8 Independence Layout", "400241", 6.4512, 7.5198, "premium"},
	{"Enugu South", "Enugu", "New Haven", "21 Ogui Road", "400252", 6.4601, 7.5102, "mid"},
	{"Enugu South", "Enugu", "Achara Layout", "10 Achara Road", "400252", 6.4389, 7.5312, "mid"},
	{"Enugu East", "Enugu", "Abakpa Nike", "14 Nike Lake Road", "400102", 6.4823, 7.5612, "budget"},
	{"Isi-Uzo", "Enugu", "Ikem", "5 Ikem Road", "402101", 6.6101, 7.7812, "budget"},

	// ─────────────────────────────────────────────
	// BENIN CITY — Edo State
	// ─────────────────────────────────────────────
	{"Oredo", "Edo", "GRA", "12 Sapele Road", "300241", 6.3350, 5.6037, "premium"},
	{"Oredo", "Edo", "GRA", "4 Oba Akenzua Road", "300241", 6.3412, 5.6101, "premium"},
	{"Ikpoba-Okha", "Edo", "New Benin", "25 Airport Road", "300252", 6.3512, 5.6282, "mid"},
	{"Egor", "Edo", "Ugbowo", "31 Ugbowo Road", "300271", 6.3701, 5.6189, "mid"},
	{"Ovia North-East", "Edo", "Ekiadolor", "9 Benin-Sapele Road", "300281", 6.4012, 5.5289, "budget"},
	{"Ikpoba-Okha", "Edo", "Uselu", "17 Uselu Road", "300252", 6.3812, 5.6312, "budget"},

	// ─────────────────────────────────────────────
	// CALABAR — Cross River State
	// ─────────────────────────────────────────────
	{"Calabar Municipal", "Cross River", "GRA", "9 Mary Slessor Avenue", "540241", 4.9701, 8.3412, "premium"},
	{"Calabar Municipal", "Cross River", "State Housing", "22 IBB Way", "540241", 4.9651, 8.3501, "mid"},
	{"Calabar Municipal", "Cross River", "Satellite Town", "15 Marian Road", "540271", 4.9512, 8.3612, "mid"},
	{"Calabar South", "Cross River", "Watt Market", "7 Esuk Utan Street", "540221", 4.9389, 8.3289, "budget"},
	{"Calabar South", "Cross River", "Big Qua Town", "33 Ekpo Abasi Road", "540221", 4.9301, 8.3188, "budget"},

	// ─────────────────────────────────────────────
	// KADUNA STATE
	// ─────────────────────────────────────────────
	{"Kaduna North", "Kaduna", "Malali GRA", "8 Ali Akilu Road", "800241", 10.5412, 7.4412, "premium"},
	{"Kaduna South", "Kaduna", "Barnawa GRA", "14 Kachia Road", "800271", 10.5012, 7.4501, "mid"},
	{"Kaduna South", "Kaduna", "Rigasa", "22 Zaria Road", "800271", 10.5189, 7.4289, "mid"},
	{"Kaduna North", "Kaduna", "Television", "11 Television Road", "800241", 10.5301, 7.4612, "budget"},
	{"Chikun", "Kaduna", "Kabala West", "5 Kabala Road", "800107", 10.5621, 7.4189, "budget"},

	// ─────────────────────────────────────────────
	// OWERRI — Imo State
	// ─────────────────────────────────────────────
	{"Owerri Municipal", "Imo", "New Owerri", "12 Douglas Road", "460241", 5.4836, 7.0362, "premium"},
	{"Owerri Municipal", "Imo", "GRA", "6 Wetheral Road", "460241", 5.4912, 7.0289, "premium"},
	{"Owerri West", "Imo", "World Bank Estate", "20 World Bank Road", "460281", 5.4712, 7.0101, "mid"},
	{"Owerri North", "Imo", "Aladinma", "9 Aladinma Road", "460001", 5.5012, 7.0412, "mid"},
	{"Owerri Municipal", "Imo", "Okigwe Road", "45 Okigwe Road", "460241", 5.5201, 7.0512, "budget"},

	// ─────────────────────────────────────────────
	// ABEOKUTA — Ogun State
	// ─────────────────────────────────────────────
	{"Abeokuta South", "Ogun", "Oke-Mosan GRA", "5 Oke-Mosan Road", "110241", 7.1512, 3.3512, "premium"},
	{"Abeokuta North", "Ogun", "Ibara GRA", "18 Ibara Road", "110241", 7.1623, 3.3391, "mid"},
	{"Obafemi-Owode", "Ogun", "Lafenwa", "29 Abeokuta-Lagos Road", "110104", 7.1312, 3.3701, "budget"},
	{"Abeokuta South", "Ogun", "Panseke", "11 Panseke Road", "110241", 7.1489, 3.3589, "budget"},

	// ─────────────────────────────────────────────
	// UYO — Akwa Ibom State
	// ─────────────────────────────────────────────
	{"Uyo", "Akwa Ibom", "GRA", "7 Abak Road", "520241", 5.0512, 7.9312, "premium"},
	{"Uyo", "Akwa Ibom", "State Housing Estate", "14 Oron Road", "520241", 5.0401, 7.9412, "mid"},
	{"Uyo", "Akwa Ibom", "Wellington Bassey Way", "33 Udo Umana Street", "520241", 5.0289, 7.9289, "mid"},
	{"Uyo", "Akwa Ibom", "Ewet Housing Estate", "21 Ring Road", "520271", 5.0612, 7.9501, "budget"},

	// ─────────────────────────────────────────────
	// DELTA STATE — Asaba & Warri
	// ─────────────────────────────────────────────
	{"Asaba", "Delta", "GRA", "9 Ibusa Road", "320241", 6.2012, 6.7312, "premium"},
	{"Asaba", "Delta", "Cable Point", "15 Anwai Road", "320241", 6.1912, 6.7412, "mid"},
	{"Warri", "Delta", "GRA", "22 Effurun-Sapele Road", "330241", 5.5312, 5.7501, "premium"},
	{"Warri South", "Delta", "Agbarho", "8 Ughelli-Warri Road", "330102", 5.5801, 5.7312, "mid"},
	{"Warri", "Delta", "Ekpan", "14 Refinery Road", "330241", 5.5612, 5.7601, "budget"},

	// ─────────────────────────────────────────────
	// OSUN STATE — Osogbo
	// ─────────────────────────────────────────────
	{"Osogbo", "Osun", "GRA", "11 Gbongan Road", "230241", 7.7689, 4.5601, "premium"},
	{"Osogbo", "Osun", "Alekuwodo", "6 Oba Adesoji Road", "230241", 7.7601, 4.5501, "mid"},
	{"Osogbo", "Osun", "Old Garage", "19 Station Road", "230241", 7.7712, 4.5701, "budget"},

	// ─────────────────────────────────────────────
	// RIVERS STATE — Outside Port Harcourt
	// ─────────────────────────────────────────────
	{"Obio-Akpor", "Rivers", "Rumuola", "8 Rumuola Road", "500258", 4.8701, 6.9912, "mid"},
	{"Obio-Akpor", "Rivers", "Rumuomasi", "22 Trans-Amadi Road", "500258", 4.8501, 6.9812, "mid"},
	{"Obio-Akpor", "Rivers", "Rukpokwu", "14 Rumuekpe Road", "500258", 4.9012, 6.9701, "budget"},

	// ─────────────────────────────────────────────
	// ANAMBRA STATE — Onitsha & Awka
	// ─────────────────────────────────────────────
	{"Onitsha North", "Anambra", "GRA", "5 New Market Road", "430241", 6.1489, 6.7889, "premium"},
	{"Onitsha South", "Anambra", "Fegge", "12 Owerri Road", "430221", 6.1389, 6.7801, "mid"},
	{"Awka South", "Anambra", "GRA", "8 Enugu Road", "420241", 6.2101, 7.0742, "premium"},
	{"Awka South", "Anambra", "Aroma Junction", "20 Zik Avenue", "420241", 6.2012, 7.0689, "mid"},
	{"Nnewi North", "Anambra", "Otolo", "15 Owerri Road", "432241", 6.0212, 6.9912, "budget"},
}

// GetLocationsByTier returns locations filtered by tier
func GetLocationsByTier(tier string) []NigerianLocation {
	var filtered []NigerianLocation
	for _, loc := range NigerianLocations {
		if loc.Tier == tier {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// GetLocationsByState returns locations in a specific state
func GetLocationsByState(state string) []NigerianLocation {
	var filtered []NigerianLocation
	for _, loc := range NigerianLocations {
		if loc.State == state {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// GetLocationsByCity returns locations in a specific city/LGA
func GetLocationsByCity(city string) []NigerianLocation {
	var filtered []NigerianLocation
	for _, loc := range NigerianLocations {
		if loc.City == city {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// GetRandomLocation returns a random location
func GetRandomLocation() NigerianLocation {
	return utils.RandomChoice(NigerianLocations)
}

// GetRandomLocationByTier returns a random location for a given tier
func GetRandomLocationByTier(tier string) NigerianLocation {
	return utils.RandomChoice(GetLocationsByTier(tier))
}
