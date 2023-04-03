namespace MoistureSensorApi;

public class SensorDataOutput
{
    public int Temperature { get; set; }
    public DateTime TemperatureTime { get; set; }
    public int Pressure { get; set; }
    public DateTime PressureTime { get; set; }
    public int Moisture { get; set; }
    public DateTime MoistureTime { get; set; }
}

