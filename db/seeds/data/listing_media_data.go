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
		"https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?auto=format&fit=crop&w=1200&q=80", // Modern apartment building exterior
		"https://images.unsplash.com/photo-1512917774080-9991f1c4c750?auto=format&fit=crop&w=1200&q=80", // Luxury living room
		"https://images.unsplash.com/photo-1502672260266-1c1ef2d93688?auto=format&fit=crop&w=1200&q=80", // Minimalist interior
		"https://images.unsplash.com/photo-1564013799919-ab600027ffc6?auto=format&fit=crop&w=1200&q=80", // Default fallback
	},

	"apartment": {
		"https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?auto=format&fit=crop&w=1200&q=80", // Modern apartment building
		"https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1200&q=80", // Cozy apartment interior
		"https://images.unsplash.com/photo-1560448204-603b3fc33ddc?auto=format&fit=crop&w=1200&q=80", // Contemporary apartment living room
		"https://images.unsplash.com/photo-1502672260266-1c1de2d96674?auto=format&fit=crop&w=1200&q=80", // Additional interior
	},

	"flat": {
		"https://images.unsplash.com/photo-1493809842364-78817add7ffb?auto=format&fit=crop&w=1200&q=80", // Modern flat interior
		"https://images.unsplash.com/photo-1502672260266-1c1ef2d93688?auto=format&fit=crop&w=1200&q=80", // Bright flat living space
		"https://images.unsplash.com/photo-1554995207-c18c203602cb?auto=format&fit=crop&w=1200&q=80", // Cozy flat bedroom
	},

	"duplex": {
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
		"https://images.unsplash.com/photo-1575517111472-7f6afd7253ca?auto=format&fit=crop&w=1200&q=80", // Cozy bungalow house
		"https://images.unsplash.com/photo-1513584684374-8bab748fbf90?auto=format&fit=crop&w=1200&q=80", // Traditional bungalow style
		"https://images.unsplash.com/photo-1602343168117-bb8ffe3e2e9f?auto=format&fit=crop&w=1200&q=80", // Bungalow with garden
		"https://images.unsplash.com/photo-1568605114967-8130f3a36994?auto=format&fit=crop&w=1200&q=80", // White bungalow
	},

	"detached_house": {
		"https://images.unsplash.com/photo-1734550028060-23cfa05bcf2a?auto=format&fit=crop&w=1200&q=80", // Aerial view of house in woods
		"https://images.unsplash.com/photo-1605276374104-6ec474f4a6fc?auto=format&fit=crop&w=1200&q=80", // Modern detached house
		"https://images.unsplash.com/photo-1600585154526-990dced4db0d?auto=format&fit=crop&w=1200&q=80", // Detached house with pool
	},

	"semi_detached": {
		"https://images.unsplash.com/photo-1605276373956-27c4bd2a5aa2?auto=format&fit=crop&w=1200&q=80", // Semi-detached houses
		"https://images.unsplash.com/photo-1582268611958-ebfd161ef9cf?auto=format&fit=crop&w=1200&q=80", // Paired residential homes
		"https://images.unsplash.com/photo-1572120360610-d971b9d7767c?auto=format&fit=crop&w=1200&q=80", // Duplex-style semi-detached
		"https://images.unsplash.com/photo-1605276374104-162f150bf228?auto=format&fit=crop&w=1200&q=80", // Brick semi-detached
	},

	"terraced": {
		"https://images.unsplash.com/photo-1586105449897-20b5efeb3233?auto=format&fit=crop&w=1200&q=80", // Row of terraced houses
		"https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?auto=format&fit=crop&w=1200&q=80", // Townhouse style terraced
		"https://images.unsplash.com/photo-1512917774080-9991f1c4c750?auto=format&fit=crop&w=1200&q=80", // Modern terraced homes
		"https://images.unsplash.com/photo-1515263487990-61b07816b324?auto=format&fit=crop&w=1200&q=80", // Colorful terraced
	},

	"townhouse": {
		"https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?auto=format&fit=crop&w=1200&q=80", // Modern townhouse
		"https://images.unsplash.com/photo-1512917774080-9991f1c4c750?auto=format&fit=crop&w=1200&q=80", // Luxury townhouse interior
		"https://images.unsplash.com/photo-1605276374104-6ec474f4a6fc?auto=format&fit=crop&w=1200&q=80", // Contemporary townhouse
		"https://images.unsplash.com/photo-1480074568708-e7b720bb3f09?auto=format&fit=crop&w=1200&q=80", // Classic townhouse exterior
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
		"https://images.unsplash.com/photo-1613490908575-9b2f6fb39e3f?auto=format&fit=crop&w=1200&q=80", // Large mansion estate
	},

	"estate": {
		"https://images.unsplash.com/photo-1605276374104-6ec474f4a6fc?auto=format&fit=crop&w=1200&q=80", // Large country estate
		"https://images.unsplash.com/photo-1600585154526-990dced4db0d?auto=format&fit=crop&w=1200&q=80", // Estate with grounds
		"https://images.unsplash.com/photo-1613977257592-4871e5fcd7c4?auto=format&fit=crop&w=1200&q=80", // Historic estate property
		"https://images.unsplash.com/photo-1588880331179-bc9b93a8cb65?auto=format&fit=crop&w=1200&q=80", // Vast estate aerial
	},

	"office_space": {
		"https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&w=1200&q=80", // Modern office workspace
		"https://images.unsplash.com/photo-1497366811353-6870744d5932?auto=format&fit=crop&w=1200&q=80", // Corporate office interior
		"https://images.unsplash.com/photo-1587574293340-e0011c4e8ecf?auto=format&fit=crop&w=1200&q=80", // Open plan office
		"https://images.unsplash.com/photo-1524758631624-e2822e304c36?auto=format&fit=crop&w=1200&q=80", // Office desk setup
	},

	"retail_space": {
		"https://images.unsplash.com/photo-1441986300917-64674bd600d8?auto=format&fit=crop&w=1200&q=80", // Modern retail store
		"https://images.unsplash.com/photo-1567401893414-76b7b1e5a7a5?auto=format&fit=crop&w=1200&q=80", // Boutique shop interior
		"https://images.unsplash.com/photo-1519567770579-c2fc5436bcf1?auto=format&fit=crop&w=1200&q=80", // Fashion retail space
	},

	"warehouse": {
		"https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?auto=format&fit=crop&w=1200&q=80", // Industrial warehouse
		"https://images.unsplash.com/photo-1553413077-190dd305871c?auto=format&fit=crop&w=1200&q=80", // Large warehouse interior
		"https://images.unsplash.com/photo-1566576912329-58c7b413f331?auto=format&fit=crop&w=1200&q=80", // Modern storage facility
	},

	"industrial": {
		"https://images.unsplash.com/photo-1581091226033-d5c48150dbaa?auto=format&fit=crop&w=1200&q=80", // Industrial building exterior
		"https://images.unsplash.com/photo-1581092335871-4c7ff3f832b1?auto=format&fit=crop&w=1200&q=80", // Manufacturing facility
		"https://images.unsplash.com/photo-1586528116493-4ce1c2b9e857?auto=format&fit=crop&w=1200&q=80", // Industrial workspace
		"https://images.unsplash.com/photo-1581091226825-a6a2a5aee158?auto=format&fit=crop&w=1200&q=80", // Factory interior
	},

	"mixed_use": {
		"https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?auto=format&fit=crop&w=1200&q=80", // Mixed commercial-residential
		"https://images.unsplash.com/photo-1512917774080-9991f1c4c750?auto=format&fit=crop&w=1200&q=80", // Modern mixed-use development
		"https://images.unsplash.com/photo-1577495508048-b635879837f1?auto=format&fit=crop&w=1200&q=80", // Urban mixed-use building
		"https://images.unsplash.com/photo-1486406146926-c627a92ad1ab?auto=format&fit=crop&w=1200&q=80", // City block mixed-use
	},

	"event_space": {
		"https://images.unsplash.com/photo-1519167758481-83f550bb49b3?auto=format&fit=crop&w=1200&q=80", // Elegant event hall
		"https://images.unsplash.com/photo-1464368081811-174160b21fd8?auto=format&fit=crop&w=1200&q=80", // Wedding reception venue
		"https://images.unsplash.com/photo-1511795409834-ef04bbd61622?auto=format&fit=crop&w=1200&q=80", // Conference event space
	},

	"co_working_space": {
		"https://images.unsplash.com/photo-1522071820081-009f0129c71c?auto=format&fit=crop&w=1200&q=80", // Modern co-working area
		"https://images.unsplash.com/photo-1557804506-669a67965ba0?auto=format&fit=crop&w=1200&q=80", // Collaborative workspace
		"https://images.unsplash.com/photo-1497366754035-f200968a6e72?auto=format&fit=crop&w=1200&q=80", // Creative co-working space
	},
}