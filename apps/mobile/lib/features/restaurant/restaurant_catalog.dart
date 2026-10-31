class RestaurantPlace {
  const RestaurantPlace({
    required this.name,
    required this.address,
    required this.lat,
    required this.lng,
    required this.placeId,
  });

  final String name;
  final String address;
  final double lat;
  final double lng;
  final String placeId;
}

const restaurantCatalog = [
  RestaurantPlace(
    name: '南城小馆',
    address: '上海市黄浦区外滩',
    lat: 31.240,
    lng: 121.490,
    placeId: 'poi-nancheng',
  ),
  RestaurantPlace(
    name: '江边面馆',
    address: '上海市虹口区',
    lat: 31.265,
    lng: 121.505,
    placeId: 'poi-jiangbian',
  ),
  RestaurantPlace(
    name: '周末咖啡',
    address: '杭州市西湖区',
    lat: 30.259,
    lng: 120.130,
    placeId: 'poi-weekend-coffee',
  ),
  RestaurantPlace(
    name: '巷口烧烤',
    address: '成都市锦江区',
    lat: 30.657,
    lng: 104.066,
    placeId: 'poi-xiangkou',
  ),
];

List<RestaurantPlace> searchRestaurants(String query) {
  final needle = query.trim().toLowerCase();
  if (needle.isEmpty) return restaurantCatalog;
  return [
    for (final place in restaurantCatalog)
      if (place.name.toLowerCase().contains(needle) ||
          place.address.toLowerCase().contains(needle))
        place,
  ];
}
