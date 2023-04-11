namespace MoistureSensorApi;

public class SensorDataOutput
{
    public int Temperature { get; set; }
    public DateTime TemperatureTime { get; set; }
    public int Humidity { get; set; }
    public DateTime HumidityTime { get; set; }
    public int Pressure { get; set; }
    public DateTime PressureTime { get; set; }
    public int SoilMoisture { get; set; }
    public DateTime SoilMoistureTime { get; set; }
    public int Light { get; set; }
    public DateTime LightTime { get; set; }
}

