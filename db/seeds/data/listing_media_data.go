package data

// ListingSeedImageLinksByPropertyType lets you inject custom image references for seed media.
//
// Accepted values:
//   - Absolute URLs (e.g., https://images.unsplash.com/...)
//   - Object keys (e.g., listings/seed/apartment/apartment-01.jpg)
//
// All images are from Unsplash's free collection. Please check individual image licenses
// for attribution requirements.
var ListingSeedImageLinksByPropertyType = map[string][]string{
	"default": {
		"https://images.unsplash.com/photo-1564013799919-ab600027ffc6?auto=format&fit=crop&w=1200&q=80", // Default fallback
		"https://a0.muscache.com/im/pictures/miso/Hosting-749776922596274302/original/c079ad3f-b473-45d2-97fa-478d573abd80.jpeg?im_w=1200",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1542042289902115401/original/617d6b9d-2a2b-4da9-9b1c-709326198830.jpeg?im_w=720",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1541828387283033250/original/8650d019-caea-467a-9dec-3a31ed040b6c.jpeg?im_w=720",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1526788674678023273/original/0227013b-9cbc-429e-a06d-48911f60270d.jpeg",
		"https://a0.muscache.com/im/pictures/a70d0c5c-762a-4495-b4b3-e3ba38a28dbd.jpg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-U3RheVN1cHBseUxpc3Rpbmc6MTQ4MTM3MjU2MjAzMjQ0NjI5MA==/original/7e5474cf-836a-4f0f-bfad-94152e218e81.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1613067641206691501/original/72717be9-fce4-4313-ae87-dd7c11aeb8a8.jpeg",
		"https://a0.muscache.com/im/pictures/airflow/Hosting-1547391315451629450/original/d1846d36-86fa-446a-99c3-dc601ec5ee33.jpg",
		"https://a0.muscache.com/im/pictures/miso/Hosting-843234273787663251/original/e85ba12b-cdc9-46bd-a694-097761bf9fa9.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1358598794506367933/original/ef76e650-efc6-4c42-bcd9-6a128c443c2e.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1511441273485156102/original/3e671427-6f43-4f02-89bf-799e5a3fcde5.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1280537358324829403/original/bf2adea5-b43c-4a16-8f21-ba4ca1d05245.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-U3RheVN1cHBseUxpc3Rpbmc6MTIzODk5MjAxMzcwNDQ2MjI1Mw%3D%3D/original/9a26a56d-a6eb-4943-938c-00d201e0be99.jpeg",
		"https://a0.muscache.com/im/pictures/cff2e560-fc4b-425f-bfab-161003cd1c70.jpg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1382004993297643411/original/c8502668-aa32-474f-b8cf-703d1f11d7bf.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1484140091189453958/original/f4334918-57c6-4a8c-9e96-0b6ac33284b4.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1389900836536836310/original/8cb36884-023a-4827-b74d-7fb9becfad9d.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1418764994498628333/original/0facbff0-96fe-4b71-8c9d-de1352ae58c5.png",
	},

	"apartment": {
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1613067641206691501/original/72717be9-fce4-4313-ae87-dd7c11aeb8a8.jpeg",
		"https://a0.muscache.com/im/pictures/airflow/Hosting-1547391315451629450/original/d1846d36-86fa-446a-99c3-dc601ec5ee33.jpg",
		"https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?auto=format&fit=crop&w=1200&q=80",    // Modern apartment building
		"https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1200&q=80", // Cozy apartment interior
		"https://images.unsplash.com/photo-1560448204-603b3fc33ddc?auto=format&fit=crop&w=1200&q=80",    // Contemporary apartment living room

	},

	"flat": {
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1389900836536836310/original/8cb36884-023a-4827-b74d-7fb9becfad9d.jpeg",
		"https://a0.muscache.com/im/pictures/hosting/Hosting-1418764994498628333/original/0facbff0-96fe-4b71-8c9d-de1352ae58c5.png",
		"https://images.unsplash.com/photo-1493809842364-78817add7ffb?auto=format&fit=crop&w=1200&q=80", // Modern flat interior
		"https://images.unsplash.com/photo-1502672260266-1c1ef2d93688?auto=format&fit=crop&w=1200&q=80", // Bright flat living space
		"https://images.unsplash.com/photo-1554995207-c18c203602cb?auto=format&fit=crop&w=1200&q=80",    // Cozy flat bedroom
	},

	"duplex": {
		"https://a0.muscache.com/im/pictures/hosting/Hosting-U3RheVN1cHBseUxpc3Rpbmc6MTIzODk5MjAxMzcwNDQ2MjI1Mw%3D%3D/original/9a26a56d-a6eb-4943-938c-00d201e0be99.jpeg",
		"https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?auto=format&fit=crop&w=1200&q=80", // Modern duplex interior with stairs
		"https://images.unsplash.com/photo-1600566753190-17f0baa2a6c3?auto=format&fit=crop&w=1200&q=80", // Luxury duplex living room
		"https://images.unsplash.com/photo-1600047509807-ba8f99d2cdde?auto=format&fit=crop&w=1200&q=80", // Duplex with spiral staircase
		"https://images.unsplash.com/photo-1574362848149-11496d93a7c7?auto=format&fit=crop&w=1200&q=80", // Additional duplex
	},

	"penthouse": {
		"https://images.unsplash.com/photo-1600607688969-a5bfcd646154?auto=format&fit=crop&w=1200&q=80", // Luxury penthouse with city view
		"https://images.unsplash.com/photo-1600607687920-4e2a09cf159d?auto=format&fit=crop&w=1200&q=80", // Rooftop penthouse terrace
		"https://images.unsplash.com/photo-1512918728675-ed5a9ecdebfd?auto=format&fit=crop&w=1200&q=80", // High-end penthouse
	},

	"studio": {
		"https://images.unsplash.com/photo-1502672260266-1c1ef2d93688?auto=format&fit=crop&w=1200&q=80", // Small studio apartment
		"https://images.unsplash.com/photo-1536376072261-38c75010e6c9?auto=format&fit=crop&w=1200&q=80", // Modern studio workspace
		"https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1200&q=80", // Compact studio layout
	},

	"house": {
		"https://images.unsplash.com/photo-1518780664697-55e3ad937233?auto=format&fit=crop&w=1200&q=80", // Suburban house exterior
		"https://images.unsplash.com/photo-1564013799919-ab600027ffc6?auto=format&fit=crop&w=1200&q=80", // Beautiful modern house
		"https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=1200&q=80", // Contemporary house design
		"https://images.unsplash.com/photo-1600596542815-ffad4c1539a9?auto=format&fit=crop&w=1200&q=80", // Modern hillside house
	},

	"bungalow": {
		"https://images.unsplash.com/photo-1513584684374-8bab748fbf90?auto=format&fit=crop&w=1200&q=80", // Traditional bungalow style
		"https://images.unsplash.com/photo-1602343168117-bb8ffe3e2e9f?auto=format&fit=crop&w=1200&q=80", // Bungalow with garden
		"https://images.unsplash.com/photo-1568605114967-8130f3a36994?auto=format&fit=crop&w=1200&q=80", // White bungalow
	},

	"detached_house": {
		"https://images.unsplash.com/photo-1600585154526-990dced4db0d?auto=format&fit=crop&w=1200&q=80", // Detached house with pool
	},

	"semi_detached": {
		"https://images.unsplash.com/photo-1582268611958-ebfd161ef9cf?auto=format&fit=crop&w=1200&q=80", // Paired residential homes
		"https://images.unsplash.com/photo-1572120360610-d971b9d7767c?auto=format&fit=crop&w=1200&q=80", // Duplex-style semi-detached
	},

	"villa": {
		"https://images.unsplash.com/photo-1613977257363-707ba9348227?auto=format&fit=crop&w=1200&q=80", // Luxury villa with pool
		"https://images.unsplash.com/photo-1613490493576-7fde63acd811?auto=format&fit=crop&w=1200&q=80", // Mediterranean style villa
		"https://images.unsplash.com/photo-1600596542815-ffad4c1539a9?auto=format&fit=crop&w=1200&q=80", // Modern hillside villa
		"https://images.unsplash.com/photo-1580587771525-78b9dba3b914?auto=format&fit=crop&w=1200&q=80", // White luxury villa
	},

	"mansion": {
		"https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=1200&q=80", // Grand mansion exterior
		"https://images.unsplash.com/photo-1600607688969-a5bfcd646154?auto=format&fit=crop&w=1200&q=80", // Luxury mansion interior
		"https://images.unsplash.com/photo-1600566753190-17f0baa2a6c3?auto=format&fit=crop&w=1200&q=80", // Elegant mansion hall
	},

	"office_space": {
		"https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&w=1200&q=80", // Modern office workspace
		"https://images.unsplash.com/photo-1524758631624-e2822e304c36?auto=format&fit=crop&w=1200&q=80", // Office desk setup
	},

	"retail_space": {
		"https://images.unsplash.com/photo-1441986300917-64674bd600d8?auto=format&fit=crop&w=1200&q=80", // Modern retail store
		"https://images.unsplash.com/photo-1567401893414-76b7b1e5a7a5?auto=format&fit=crop&w=1200&q=80", // Boutique shop interior
	},

	"warehouse": {
		"https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?auto=format&fit=crop&w=1200&q=80", // Industrial warehouse
		"https://images.unsplash.com/photo-1553413077-190dd305871c?auto=format&fit=crop&w=1200&q=80",    // Large warehouse interior
	},

	"industrial": {
		"https://images.unsplash.com/photo-1581091226033-d5c48150dbaa?auto=format&fit=crop&w=1200&q=80", // Industrial building exterior
		"https://images.unsplash.com/photo-1581091226825-a6a2a5aee158?auto=format&fit=crop&w=1200&q=80", // Factory interior
	},
	"event_space": {
		"https://images.unsplash.com/photo-1519167758481-83f550bb49b3?auto=format&fit=crop&w=1200&q=80", // Elegant event hall
		"https://images.unsplash.com/photo-1511795409834-ef04bbd61622?auto=format&fit=crop&w=1200&q=80", // Conference event space
	},

	"co_working_space": {
		"https://images.unsplash.com/photo-1522071820081-009f0129c71c?auto=format&fit=crop&w=1200&q=80", // Modern co-working area
		"https://images.unsplash.com/photo-1557804506-669a67965ba0?auto=format&fit=crop&w=1200&q=80",    // Collaborative workspace
		"https://images.unsplash.com/photo-1497366754035-f200968a6e72?auto=format&fit=crop&w=1200&q=80", // Creative co-working space
	},
}
