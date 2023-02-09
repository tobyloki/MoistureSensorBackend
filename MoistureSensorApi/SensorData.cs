namespace MoistureSensorApi;

public class SensorData
{
    public string DeviceId { get; set; } = null!;

    public int Temperature { get; set; }

    public int Pressure { get; set; }

    public int Moisture { get; set; }
}

