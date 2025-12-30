package data

import "hauslet/db/seeds/utils"

// NigerianNames contains common Nigerian first and last names
var (
	NigerianFirstNames = []string{
		// Yoruba names
		"Adebayo", "Adewale", "Adesola", "Ayomide", "Bolaji",
		"Chioma", "Damilola", "Funmilayo", "Kehinde", "Olamide",
		"Oluwaseun", "Temitope", "Titilayo", "Yetunde", "Folake",

		// Igbo names
		"Chukwuemeka", "Chidinma", "Ngozi", "Obinna", "Uchenna",
		"Ifeanyi", "Amarachi", "Chinedu", "Ebere", "Nkechi",
		"Obioma", "Onyinyechi", "Somtochukwu", "Nnamdi", "Adaeze",

		// Hausa names
		"Abubakar", "Fatima", "Hassan", "Ibrahim", "Khadija",
		"Musa", "Sadiya", "Usman", "Zainab", "Aliyu",
		"Hadiza", "Bilkisu", "Yusuf", "Amina", "Ahmad",

		// Modern/Neutral
		"David", "Emmanuel", "Grace", "John", "Mary",
		"Michael", "Peter", "Samuel", "Victoria", "Daniel",
	}

	NigerianLastNames = []string{
		// Yoruba surnames
		"Adeyemi", "Ajayi", "Bakare", "Falana", "Gbadamosi",
		"Ogunleye", "Oluwole", "Oladipo", "Williams", "Adebisi",
		"Odunsi", "Adekunle", "Olabode", "Olatunji", "Babatunde",

		// Igbo surnames
		"Okafor", "Okeke", "Nwankwo", "Onuoha", "Azubuike",
		"Eze", "Okonkwo", "Nwosu", "Ihejirika", "Obiorah",
		"Ugochukwu", "Mbanefo", "Chukwuma", "Ikechukwu", "Obiageli",

		// Hausa surnames
		"Bello", "Sani", "Umar", "Mohammed", "Garba",
		"Shehu", "Abubakar", "Musa", "Yusuf", "Suleiman",

		// Other Nigerian surnames
		"Obi", "Eze", "Musa", "Johnson", "Okoro",
		"Adamu", "Hassan", "Lawal", "Ibrahim", "Aliyu",
	}

	NigerianBusinessNames = []string{
		"Prime Properties Nigeria",
		"Cityscape Realty",
		"Lagos Homes Limited",
		"Abuja Property Management",
		"Heritage Real Estate",
		"Royal Estates Nigeria",
		"Urban Living Solutions",
		"Premier Property Partners",
		"Nova Realty Services",
		"Landmark Properties",
		"Gateway Homes & Estates",
		"Metropolitan Property Group",
		"Elite Property Consultants",
		"Zenith Real Estate",
		"Crown Properties Limited",
	}
)

// GenerateFullName creates a random full name
func GenerateFullName() string {
	firstName := utils.RandomChoice(NigerianFirstNames)
	lastName := utils.RandomChoice(NigerianLastNames)
	return firstName + " " + lastName
}
