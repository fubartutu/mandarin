// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package shopping

// DefaultCatalog is the item catalogue a new household starts with, keyed by
// the default category titles in DefaultCategories. Anything the household
// types that isn't here is added to the catalogue automatically.
var DefaultCatalog = map[string][]string{
	"Fruit & Veg": {
		"Apples", "Avocados", "Baby spinach", "Bananas", "Basil", "Blueberries", "Broccoli",
		"Brown onions", "Capsicum", "Carrots", "Cauliflower", "Celery", "Cherry tomatoes",
		"Coriander", "Corn", "Cucumber", "Garlic", "Ginger", "Grapes", "Green beans", "Kiwi fruit",
		"Lemons", "Lettuce", "Limes", "Mandarins", "Mangoes", "Mushrooms", "Oranges", "Parsley",
		"Pears", "Potatoes", "Pumpkin", "Red onions", "Rocket", "Salad mix", "Spring onions",
		"Strawberries", "Sweet potatoes", "Tomatoes", "Watermelon", "Zucchini",
	},
	"Bakery": {
		"Bread", "Bread rolls", "Croissants", "English muffins", "Hot cross buns", "Raisin toast",
		"Sourdough", "Tortillas", "Wraps", "Crumpets",
	},
	"Meat & Seafood": {
		"Bacon", "Beef mince", "Chicken breast", "Chicken drumsticks", "Chicken mince",
		"Chicken thighs", "Fish fillets", "Lamb chops", "Lamb roast", "Pork chops", "Pork mince",
		"Prawns", "Salmon", "Sausages", "Steak", "Whole chicken",
	},
	"Deli": {
		"Dips", "Feta", "Ham", "Hummus", "Olives", "Salami", "Shaved turkey", "Sliced chicken",
	},
	"Dairy & Eggs": {
		"Butter", "Cheddar cheese", "Cream", "Cream cheese", "Custard", "Eggs", "Grated cheese",
		"Greek yoghurt", "Halloumi", "Milk", "Mozzarella", "Parmesan", "Sliced cheese",
		"Sour cream", "Yoghurt",
	},
	"Fridge": {
		"Fresh pasta", "Gnocchi", "Orange juice", "Pastry", "Tofu",
	},
	"Frozen": {
		"Chips", "Fish fingers", "Frozen berries", "Frozen corn", "Frozen mixed vegetables",
		"Frozen pizza", "Frozen peas", "Ice", "Ice cream", "Puff pastry", "Ice blocks",
	},
	"Pantry": {
		"Baked beans", "Baking powder", "Basmati rice", "Bicarb soda", "Breadcrumbs",
		"Brown sugar", "Canned chickpeas", "Canned tomatoes", "Canned tuna", "Chicken stock",
		"Coconut milk", "Cornflour", "Curry paste", "Dried pasta", "Flour", "Gravy",
		"Honey", "Jam", "Kidney beans", "Lentils", "Mayonnaise", "Mustard", "Noodles",
		"Olive oil", "Pasta sauce", "Peanut butter", "Pepper", "Rice", "Salt", "Soy sauce",
		"Spaghetti", "Sugar", "Sweet chilli sauce", "Tomato paste", "Tomato sauce",
		"Vanilla essence", "Vegemite", "Vegetable oil", "Vinegar",
	},
	"Breakfast": {
		"Cereal", "Coffee", "Coffee pods", "Muesli", "Oats", "Porridge", "Tea bags", "Weet-Bix",
	},
	"Snacks & Sweets": {
		"Biscuits", "Chocolate", "Corn chips", "Crackers", "Lollies", "Muesli bars", "Nuts",
		"Popcorn", "Potato chips", "Rice crackers", "Shapes", "Tim Tams",
	},
	"Drinks": {
		"Apple juice", "Cordial", "Kombucha", "Mineral water", "Soft drink", "Sparkling water",
	},
	"Household & Cleaning": {
		"Aluminium foil", "Baking paper", "Batteries", "Bin bags", "Cling wrap", "Dishwashing liquid",
		"Dishwasher tablets", "Laundry liquid", "Light globes", "Paper towel", "Sponges",
		"Spray and wipe", "Tissues", "Toilet cleaner", "Toilet paper", "Zip lock bags",
	},
	"Health & Beauty": {
		"Band-aids", "Conditioner", "Deodorant", "Hand wash", "Paracetamol", "Razors",
		"Shampoo", "Soap", "Sunscreen", "Toothbrushes", "Toothpaste",
	},
	"Baby & Kids": {
		"Baby wipes", "Lunchbox snacks", "Nappies", "Squeeze pouches",
	},
	"Pets": {
		"Cat food", "Cat litter", "Dog food", "Pet treats",
	},
	"Clothing & Shoes": {
		"School shoes", "School socks", "Socks", "Underwear",
	},
}
